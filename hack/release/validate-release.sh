#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

source ./hack/release/release-lib.sh

metadata_file="${1:-.github/release/handoff.json}"

release_load_metadata "$metadata_file"
operator_version="$OPERATOR_VERSION"
previous_operator_version="$PREVIOUS_OPERATOR_VERSION"
chart_version="$CHART_VERSION"
runner_k6_version="$RUNNER_K6_VERSION"

required_files=(
  Makefile
  bundle.yaml
  charts/k6-operator/README.md
  charts/k6-operator/Chart.yaml
  charts/k6-operator/values.yaml
  charts/k6-operator/values.schema.json
  docs/versioning.md
  e2e/latest
  e2e/latest/Deployment-k6-operator-controller-manager.yml
)
for required_file in "${required_files[@]}"; do
  if [[ ! -e "$required_file" ]]; then
    echo "Required release file is missing: $required_file" >&2
    exit 1
  fi
done

release_validate_operator_version "$operator_version" "operator version"
release_validate_operator_version "$previous_operator_version" "previous operator version"
release_validate_chart_version "$chart_version" "chart version"

if ! git rev-parse -q --verify "refs/tags/${previous_operator_version}" > /dev/null; then
  echo "Previous operator tag does not exist: ${previous_operator_version}" >&2
  exit 1
fi

grep -qxF "$(release_makefile_version_line "$operator_version")" Makefile
grep -qxF "$(release_chart_appversion_line "$operator_version")" charts/k6-operator/Chart.yaml
grep -qxF "$(release_chart_version_line "$chart_version")" charts/k6-operator/Chart.yaml
grep -qxF "$(release_values_tag_line "$operator_version")" charts/k6-operator/values.yaml
grep -qF "ghcr.io/grafana/k6-operator:controller-${operator_version}" bundle.yaml
grep -qF "ghcr.io/grafana/k6-operator:controller-${operator_version}" e2e/latest/Deployment-k6-operator-controller-manager.yml

if ! grep -qxF "$(release_versioning_row "$operator_version" "$runner_k6_version")" docs/versioning.md; then
  echo "docs/versioning.md does not contain the expected row for ${operator_version}" >&2
  exit 1
fi

echo "Release metadata validated for ${operator_version}"
