#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=hack/release/release-lib.sh
source "${script_dir}/release-lib.sh"

# Validates the dispatch inputs and fails fast, before any image is built, if
# the release tag or release-prep branch already exists. Exports the resolved
# values to $GITHUB_ENV for later steps.
if [[ "${GITHUB_REF_NAME:-}" != "main" ]]; then
  release_error "Prepare release must be triggered from main; got ${GITHUB_REF_NAME:-<unset>}"
  exit 1
fi

release_validate_inputs

git fetch --tags --force
if git rev-parse -q --verify "refs/tags/${OPERATOR_VERSION}" > /dev/null; then
  release_error "Tag ${OPERATOR_VERSION} already exists"
  exit 1
fi

branch="release-${OPERATOR_VERSION}/prepare"
if git ls-remote --exit-code --heads origin "$branch" > /dev/null 2>&1; then
  release_error "Branch $branch already exists"
  exit 1
fi

release_set_env OPERATOR_VERSION "$OPERATOR_VERSION"
release_set_env CHART_VERSION "$CHART_VERSION"
release_set_env RELEASE_BRANCH "$branch"
