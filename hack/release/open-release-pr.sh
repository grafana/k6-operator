#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

# shellcheck source=hack/release/release-lib.sh
source ./hack/release/release-lib.sh

# Commits the generated release artifacts onto the release-prep branch and opens
# the release PR. Requires GITHUB_TOKEN in the environment for `gh`.
release_require_env OPERATOR_VERSION
release_require_env CHART_VERSION
release_require_env RELEASE_BRANCH

branch="$RELEASE_BRANCH"

# check-prepare-preconditions.sh already verified the branch does not exist;
# `git push -u` below fails naturally if it appeared in the meantime.

# Check label existence via API.
repo="${GITHUB_REPOSITORY:-$(gh repo view --json nameWithOwner --jq '.nameWithOwner')}"
if ! gh api "repos/${repo}/labels/${RELEASE_PR_LABEL}" --silent 2> /dev/null; then
  release_error "Required label '${RELEASE_PR_LABEL}' does not exist in ${repo}"
  exit 1
fi

git checkout -b "$branch"
# Stage every change produced by `make release-prepare` (new, modified, and
# deleted files). The CI checkout is clean, so the only changes are generated
# release artifacts; a curated list would silently drop any generator output
# not listed here.
git add -A

if git diff --cached --quiet; then
  release_error "Release preparation produced no changes"
  exit 1
fi

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git commit -m "release: prepare ${OPERATOR_VERSION}"
git push -u origin "$branch"

pr_body_file="$(mktemp)"
cat > "$pr_body_file" <<EOF
Automated release-prep PR for ${OPERATOR_VERSION}.

- Operator version: ${OPERATOR_VERSION}
- Chart version: ${CHART_VERSION}
- Runner k6 version: $(jq -r '.runner_k6_version' .github/release/handoff.json)

Merging this PR will trigger the finalize workflow, which promotes latest image tags, creates the git tag, and creates a draft GitHub release.
EOF

gh pr create \
  --title "release: prepare ${OPERATOR_VERSION}" \
  --base main \
  --head "$branch" \
  --label "$RELEASE_PR_LABEL" \
  --body-file "$pr_body_file"
