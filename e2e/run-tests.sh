#!/bin/bash

# Source .env for CLOUD_TOKEN, PROJECT_ID, etc.
if [ -f "$(dirname "$0")/.env" ]; then
  set -a
  source "$(dirname "$0")/.env"
  set +a
fi

show_help() {
  echo "Usage: $(basename $0) [-h] [-t GHCR_IMAGE_TAG] [-i IMAGE] [-p TEST_NAME]"
  echo "Options:"
  echo "  -h      Display this help message."
  echo "  -t      Existing tag of ghcr.io/grafana/k6-operator image."
  echo "  -i      Arbitrary Docker image for k6-operator."
  echo "  -u      Update manifests in latest folder."
  echo "  -p      Pick one test (folder) to run separately."
  exit 0
}

exec_test() {
  echo "Executing test $TEST_NAME"
  cd $TEST_NAME/manifests
  for f in *.yaml; do envsubst '$CLOUD_TOKEN $PROJECT_ID' < $f > out && mv out $f; done
  cd .. # back to $TEST_NAME
  if ! ../k6 run test.js ; then
    echo "Test $TEST_NAME failed"
    git checkout -- manifests/
    cd .. # back to root
    exit 1
  fi
  git checkout -- manifests/
  cd .. # back to root
}

# use GHCR latest image by default, unless -i is specified
GHCR_IMAGE_TAG=latest
IMAGE=
TEST_NAME=

while getopts ':hut:i:p:' option; do
  case "$option" in
    h) show_help
       exit
       ;;
    t) GHCR_IMAGE_TAG=$OPTARG
       ;;
    i) IMAGE=$OPTARG
       ;;
    u) UPDATE_LATEST=true
       ;;
    p) TEST_NAME=$OPTARG
       ;;
    :) printf "missing argument for -%s\n" "$OPTARG" >&2
       show_help
       exit 1
       ;;
    \?) printf "illegal option: -%s\n" "$OPTARG" >&2
       show_help
       exit 1
       ;;
  esac
done
# shift $((OPTIND - 1))

if [ -z "${IMAGE}" ]; then
  IMAGE=ghcr.io/grafana/k6-operator:$GHCR_IMAGE_TAG
fi

echo "Using k6-operator image:" $IMAGE

# Recreate kustomization.

echo "Regenerate ./latest from the bundle"

rm latest/*

# use an existing bundle.yaml if it fits the image or generate a new one
if [ "$IMAGE" = "ghcr.io/grafana/k6-operator:latest" ]; then
  cp ../bundle.yaml ./latest/bundle-to-test.yaml
  cd latest
else
  cd ../config/manager/
  kustomize edit set image controller=$IMAGE
  cd ../default
  kustomize build . > ../../e2e/latest/bundle-to-test.yaml
  cd ../../e2e/latest
fi

# We're in e2e/latest here and there is a bundle-to-test.yaml
# Split the bundle and create a kustomize

docker run --user="$(id -u)" --rm -v "${PWD}":/workdir mikefarah/yq --no-doc  -s  '.kind + "-" + .metadata.name' bundle-to-test.yaml
# since CRDs are being extracted as k6.io, without yaml in the end, rename them:
for f in $(find . -type f  -name '*.k6.io'); do mv $f ${f}.yaml; done

rm bundle-to-test.yaml
kustomize create --autodetect --recursive .

# go back to e2e/
cd ..

if [[ $UPDATE_LATEST == true ]] ; then
  echo "Latest manifests were updated."
  exit 0
fi

if [ ! -f ./k6 ]; then
  TAG=$(curl -sL "https://api.github.com/repos/grafana/xk6-environment/releases/latest" | grep -o '"tag_name": "[^"]*' | cut -d'"' -f4)
  curl -L -o k6-linux-amd64.tar.gz "https://github.com/grafana/xk6-environment/releases/download/${TAG}/k6-${TAG}-linux-amd64.tar.gz"
  tar -xzf k6-linux-amd64.tar.gz
  chmod +x k6
  ./k6 version
fi

# Run the tests.

if [ ! -z "${TEST_NAME}" ]; then
  exec_test
  exit 0
fi

tests=($(grep '^  - name:' tests.yaml | awk '{print $3}'))

for folder in "${tests[@]}"; do
    TEST_NAME=$folder
    exec_test
done
