#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=hack/release/release-lib.sh
source "${script_dir}/release-lib.sh"

# authorize_pr is the authoritative gate. It re-validates the release-prep
# conventions (merged into main, same-repo head, release-v*/prepare branch, a
# real merge commit, the release label) against the merge event, read from the
# EVENT_* env vars the workflow passes from github.event.pull_request. The merge
# is terminal, so the event payload and live API state agree; the job `if:` is
# only a cheap pre-filter.
authorize_pr() {
  local pr_number="${EVENT_PR_NUMBER:-}"

  release_require_env GITHUB_REPOSITORY
  release_require_env EVENT_MERGED
  release_require_env EVENT_BASE_REF
  release_require_env EVENT_HEAD_REF
  release_require_env EVENT_HEAD_REPO

  if [[ ! "$pr_number" =~ ^[0-9]+$ ]]; then
    release_error "pr_number must be numeric"
    exit 1
  fi

  if [[ "$EVENT_MERGED" != "true" ]]; then
    release_error "PR #${pr_number} is not merged"
    exit 1
  fi

  if [[ "$EVENT_BASE_REF" != "main" ]]; then
    release_error "PR #${pr_number} base branch is $EVENT_BASE_REF, expected main"
    exit 1
  fi

  if [[ "$EVENT_HEAD_REPO" != "$GITHUB_REPOSITORY" ]]; then
    release_error "PR #${pr_number} came from $EVENT_HEAD_REPO, expected $GITHUB_REPOSITORY"
    exit 1
  fi

  if [[ "$EVENT_HEAD_REF" != release-v*/prepare ]]; then
    release_error "PR #${pr_number} branch is $EVENT_HEAD_REF, expected release-v*/prepare"
    exit 1
  fi

  if [[ -z "${EVENT_MERGE_SHA:-}" ]]; then
    release_error "PR #${pr_number} does not have a merge commit"
    exit 1
  fi

  if ! jq -e --arg label "$RELEASE_PR_LABEL" 'index($label) != null' \
    <<< "${EVENT_LABELS:-[]}" > /dev/null; then
    release_error "PR #${pr_number} does not have the ${RELEASE_PR_LABEL} label"
    exit 1
  fi
}

validate_contents() {
  local repo_root
  local metadata_file="${1:-.github/release/handoff.json}"
  local operator_version
  local existing_tag_sha
  local expected_sha

  repo_root="$(git rev-parse --show-toplevel)"
  cd "$repo_root"

  release_require_env GITHUB_REPOSITORY
  release_require_env MERGE_COMMIT_SHA
  if [[ -z "${GITHUB_TOKEN:-}${GH_TOKEN:-}" ]]; then
    echo "GITHUB_TOKEN or GH_TOKEN is required" >&2
    exit 1
  fi
  expected_sha="$MERGE_COMMIT_SHA"

  "${script_dir}/validate-release.sh" "$metadata_file"

  release_load_metadata "$metadata_file"
  operator_version="$OPERATOR_VERSION"

  if gh release view "$operator_version" > /dev/null 2>&1; then
    # A published release is final and must never be clobbered. An existing
    # *draft* is allowed: it means a prior finalize run got as far as draft
    # creation, so a re-run can update it in place.
    if [[ "$(gh release view "$operator_version" --json isDraft --jq '.isDraft')" != "true" ]]; then
      release_error "Release $operator_version already exists and is published"
      exit 1
    fi
    echo "Draft release $operator_version already exists; it will be updated"
  fi

  # gh api prints the error body to stdout (and skips --jq) on a non-2xx
  # response, so key off its exit status rather than the captured output:
  # a missing tag must leave existing_tag_sha empty, not holding the 404 body.
  if ! existing_tag_sha="$(
    gh api "repos/${GITHUB_REPOSITORY}/git/ref/tags/${operator_version}" \
      --jq '.object.sha' 2> /dev/null
  )"; then
    existing_tag_sha=""
  fi

  if [[ -n "$existing_tag_sha" && "$existing_tag_sha" != "$expected_sha" ]]; then
    release_error "Tag $operator_version already exists at $existing_tag_sha"
    exit 1
  fi

  release_set_env OPERATOR_VERSION "$operator_version"
}

command="${1:-}"
if [[ $# -gt 0 ]]; then
  shift
fi

case "$command" in
  authorize-pr)
    authorize_pr "$@"
    ;;
  validate-contents)
    validate_contents "$@"
    ;;
  *)
    echo "Usage: $0 {authorize-pr|validate-contents} [metadata-file]" >&2
    exit 1
    ;;
esac
