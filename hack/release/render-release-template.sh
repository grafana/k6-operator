#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

# shellcheck source=hack/release/release-lib.sh
source ./hack/release/release-lib.sh

metadata_file="${1:-.github/release/handoff.json}"
template_file="${2:-.github/release/release-template.md}"
output_file="${3:-release-notes.md}"

release_load_metadata "$metadata_file"

sed \
  -e "s/{{OPERATOR_VERSION}}/${OPERATOR_VERSION}/g" \
  -e "s/{{PREVIOUS_OPERATOR_VERSION}}/${PREVIOUS_OPERATOR_VERSION}/g" \
  -e "s/{{CHART_VERSION}}/${CHART_VERSION}/g" \
  -e "s/{{RUNNER_K6_VERSION}}/${RUNNER_K6_VERSION}/g" \
  "$template_file" > "$output_file"

echo "Rendered release notes to ${output_file}"
