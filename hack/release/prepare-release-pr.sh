#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

# shellcheck source=hack/release/release-lib.sh
source ./hack/release/release-lib.sh

# Prepares the inputs for the release-prep PR: verifies the required label exists
# and renders the PR body. 
# Requires GITHUB_TOKEN in the environment for `gh`.

release_require_env OPERATOR_VERSION
release_require_env CHART_VERSION

# If label for releases is missing, fail fast.
repo="${GITHUB_REPOSITORY:-$(gh repo view --json nameWithOwner --jq '.nameWithOwner')}"
if ! gh api "repos/${repo}/labels/${RELEASE_PR_LABEL}" --silent 2> /dev/null; then
  release_error "Required label '${RELEASE_PR_LABEL}' does not exist in ${repo}"
  exit 1
fi

# Render the PR body outside the working tree so the GA workflow doesn't commit it.
body_file="${RUNNER_TEMP:-/tmp}/release-pr-body.md"
cat > "$body_file" <<EOF
Automated release-prep PR for ${OPERATOR_VERSION}.

- Operator version: ${OPERATOR_VERSION}
- Chart version: ${CHART_VERSION}
- Runner k6 version: $(jq -r '.runner_k6_version' .github/release/handoff.json)

Merging this PR will trigger the finalize workflow, which promotes latest image tags, creates the git tag, and creates a draft GitHub release.
EOF

release_set_output body_file "$body_file"
