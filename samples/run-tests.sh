#!/bin/bash

show_help() {
  echo "Usage: $(basename $0) [-h] [-t GHCR_IMAGE_TAG] [-i IMAGE] [-p TEST_NAME]"
  echo "Options:"
  echo "  -h      Display this help message"
  echo "  -t      Existing tag of ghcr.io/grafana/k6-operator image"
  echo "  -i      Arbitrary Docker image for k6-operator"
  echo "  -p      Pick one test (folder) to run separately"
  exit 0
}

exec_test() {
  echo "Executing test $TEST_NAME"
  cd $TEST_NAME/manifests
  for f in *.yaml; do envsubst '$CLOUD_TOKEN' < $f > out && mv out $f; done
  cd .. # back to $TEST_NAME
  if ! ../k6 run test.js ; then
    echo "Test $TEST_NAME failed"
    cd .. # back to root
    exit 1
  fi
  cd .. # back to root
}

# use GHCR latest image by default, unless -i is specified
GHCR_IMAGE_TAG=latest
IMAGE=
TEST_NAME=

while getopts ':ht:i:p:' option; do
  case "$option" in
    h) show_help
       exit
       ;;
    t) GHCR_IMAGE_TAG=$OPTARG
       ;;
    i) IMAGE=$OPTARG
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
  cd ../config/default 
  kustomize edit set image $IMAGE && kustomize build . > ../../samples/latest/bundle-to-test.yaml
  cd ../../samples/latest
fi

# We're in samples/latest here and there is a bundle-to-test.yaml
# Split the bundle and create a kustomize

docker run --user="1001" --rm -v "${PWD}":/workdir mikefarah/yq --no-doc  -s  '.kind + "-" + .metadata.name' bundle-to-test.yaml
# since CRDs are being extracted as k6.io, without yaml in the end, rename them:
for f in $(find . -type f  -name '*.k6.io'); do mv $f ${f}.yaml; done

rm bundle-to-test.yaml
kustomize create --autodetect --recursive .

# since CRDs are being extracted as k6.io, without yaml in the end, rename them:
for f in $(find . -type f  -name '*.k6.io'); do mv $f ${f}.yaml; done

# go back to samples/
cd ..

# TODO: add a proper build with xk6-environment (use new functionality?)
# Blocked by: https://github.com/grafana/xk6-environment/issues/16
# Right now, using the pre-built k6 binary uploaded to a branch in xk6-environment

if [ ! -f ./k6 ]; then
  wget https://github.com/grafana/xk6-environment/raw/refs/heads/fix/temp-k6-binary/bin/k6
  chmod +x ./k6
fi

# Run the tests.

if [ ! -z "${TEST_NAME}" ]; then
  exec_test
  exit 0
fi

tests=(
  "basic-testrun-1" 
  "basic-testrun-4"
  "testrun-cleanup"
  "testrun-archive"
  "init-container-volume"
  "multifile"
  "error-stage"
  "testrun-simultaneous"
  "testrun-watch-namespace"
  "testrun-cloud-output"
  "testrun-simultaneous-cloud-output"
  # "kyverno"
  # "custom-domain"
  # "browser-1"
  # cloud abort
  # plz
  )

for folder in "${tests[@]}"; do
    TEST_NAME=$folder
    exec_test
done
