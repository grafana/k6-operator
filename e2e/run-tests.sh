#!/bin/bash

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

# since CRDs are being extracted as k6.io, without yaml in the end, rename them:
for f in $(find . -type f  -name '*.k6.io'); do mv $f ${f}.yaml; done

# go back to e2e/
cd ..

if [[ $UPDATE_LATEST == true ]] ; then
  echo "Latest manifests were updated."
  exit 0
fi

# TODO: add a proper build with xk6-environment (use new functionality?)
# Blocked by: https://github.com/grafana/xk6-environment/issues/16
# Right now, using the pre-built k6 binary uploaded to a branch in xk6-environment
# Note that this is an ELF binary built for x86-64, so it will not work on other platforms.
# To build it yourself:
#   1. clone the xk6-environment repo
#   2. checkout the 0.1.0 tag
#   3. edit the 'build' target in the Makefile to use v0.13.0 of go.k6.io/xk6/cmd/xk6 (instead of latest)
#   4. `make build`
#   5. copy the k6 binary to this directory

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
  "invalid-arguments"
  "testrun-simultaneous"
  "testrun-watch-namespace"
  "testrun-watch-namespaces"
  "testrun-cloud-output"
  "testrun-simultaneous-cloud-output"
  # volume-claim
  # "kyverno"
  # "custom-domain"
  # "browser-1"
  # cloud abort
  # plz
  # ipv6
  )

for folder in "${tests[@]}"; do
    TEST_NAME=$folder
    exec_test
done
