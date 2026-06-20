#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

source ./hack/release/release-lib.sh

# The runner ships the k6 binary from the pinned base image in Dockerfile.runner,
# so the shipped k6 version is read straight from that FROM line rather than by
# running the image. Bumping k6 is a deliberate edit to Dockerfile.runner.
read_runner_k6_version() {
  local from_tag
  from_tag="$(sed -nE 's#^FROM[[:space:]]+grafana/k6:([^@[:space:]]+).*#\1#p' Dockerfile.runner | head -n 1)"
  if [[ -z "$from_tag" ]]; then
    return 1
  fi
  printf 'v%s\n' "${from_tag#v}"
}

append_versioning_row() {
  local tmp_file

  tmp_file="$(mktemp)"
  awk -F '|' -v version="$OPERATOR_VERSION" '
    {
      row_version = $2
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", row_version)
      if (row_version != version) {
        print
      }
    }
  ' \
    docs/versioning.md > "$tmp_file"
  mv "$tmp_file" docs/versioning.md

  printf '%s\n' \
    "$(release_versioning_row "$OPERATOR_VERSION" "$RUNNER_K6_VERSION")" \
    >> docs/versioning.md
}

write_release_metadata() {
  jq -n \
    --arg operator_version "$OPERATOR_VERSION" \
    --arg previous_operator_version "$PREVIOUS_OPERATOR_VERSION" \
    --arg chart_version "$CHART_VERSION" \
    --arg runner_k6_version "$RUNNER_K6_VERSION" \
    '{
      operator_version: $operator_version,
      previous_operator_version: $previous_operator_version,
      chart_version: $chart_version,
      runner_k6_version: $runner_k6_version
    }' > .github/release/handoff.json
}

release_validate_inputs

PREVIOUS_OPERATOR_VERSION="${PREVIOUS_OPERATOR_VERSION:-$(git tag --list 'v[0-9]*' --sort=-v:refname | head -n 1)}"

if [[ -z "$PREVIOUS_OPERATOR_VERSION" ]]; then
  echo "PREVIOUS_OPERATOR_VERSION is required when no previous v* tag exists" >&2
  exit 1
fi
release_validate_operator_version "$PREVIOUS_OPERATOR_VERSION" "PREVIOUS_OPERATOR_VERSION"

RUNNER_K6_VERSION="${RUNNER_K6_VERSION:-$(read_runner_k6_version)}"
if [[ -z "$RUNNER_K6_VERSION" ]]; then
  echo "Could not read the k6 version from Dockerfile.runner" >&2
  exit 1
fi

sed -i -E "s/^VERSION \?= .*/$(release_makefile_version_line "$OPERATOR_VERSION")/" Makefile
sed -i -E "s/^appVersion: .*/$(release_chart_appversion_line "$OPERATOR_VERSION")/" charts/k6-operator/Chart.yaml
sed -i -E "s/^version: .*/$(release_chart_version_line "$CHART_VERSION")/" charts/k6-operator/Chart.yaml
# Only the first `tag:` line (manager.image.tag); avoids clobbering any other tag keys.
sed -i -E "0,/^[[:space:]]*tag: .*/s//$(release_values_tag_line "$OPERATOR_VERSION")/" charts/k6-operator/values.yaml

append_versioning_row
write_release_metadata

echo "Prepared release metadata for ${OPERATOR_VERSION} with runner k6 ${RUNNER_K6_VERSION}"
