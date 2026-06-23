#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=hack/release/release-lib.sh
source "${script_dir}/release-lib.sh"

SHA256_DIGEST_REGEX='^[0-9a-f]{64}$'

# latest* must be promoted from immutable digest refs, never floating tags, so
# latest* always points at exactly the image that was just built and pushed.
require_sha256_ref() {
  local name="$1"
  local source="$2"
  local digest="${2##*@sha256:}"

  if [[ "$source" != *@sha256:* || ! "$digest" =~ $SHA256_DIGEST_REGEX ]]; then
    echo "${name} image source must be a sha256 digest ref; got $source" >&2
    exit 1
  fi
}

if [[ "$#" -ne 3 ]]; then
  echo "Usage: $0 CONTROLLER_IMAGE RUNNER_IMAGE STARTER_IMAGE" >&2
  exit 1
fi

controller_image="$1"
runner_image="$2"
starter_image="$3"

require_sha256_ref "controller" "$controller_image"
require_sha256_ref "runner" "$runner_image"
require_sha256_ref "starter" "$starter_image"

promote() {
  docker buildx imagetools create -t "${RELEASE_IMAGE_NAME}:$1" "$2"
}

promote latest "$controller_image"
promote latest-runner "$runner_image"
promote latest-starter "$starter_image"
