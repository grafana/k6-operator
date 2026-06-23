#!/usr/bin/env bash

if [[ -n "${K6_OPERATOR_RELEASE_LIB_SOURCED:-}" ]]; then
  return 0
fi
K6_OPERATOR_RELEASE_LIB_SOURCED=1

RELEASE_IMAGE_NAME="${RELEASE_IMAGE_NAME:-ghcr.io/grafana/k6-operator}"
# Keep this default in sync with the 'operator-release' literal in the
# finalize-release.yaml job `if:`, which cannot read shell variables.
RELEASE_PR_LABEL="${RELEASE_PR_LABEL:-operator-release}"
OPERATOR_VERSION_REGEX='^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'
CHART_VERSION_REGEX='^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'

# release_error emits a message as a GitHub Actions error annotation (on stdout,
# where Actions reads workflow commands) and is a no-op-safe plain print locally.
release_error() {
  echo "::error::$*"
}

# release_set_env / release_set_output append to the GitHub Actions step files
# when present, and are silent no-ops when run locally.
release_set_env() {
  if [[ -n "${GITHUB_ENV:-}" ]]; then
    printf '%s=%s\n' "$1" "$2" >> "$GITHUB_ENV"
  fi
}

release_set_output() {
  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    printf '%s=%s\n' "$1" "$2" >> "$GITHUB_OUTPUT"
  fi
}

release_require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "$name is required" >&2
    exit 1
  fi
}

release_validate_inputs() {
  release_require_env OPERATOR_VERSION
  release_require_env CHART_VERSION
  release_validate_operator_version "$OPERATOR_VERSION"
  release_validate_chart_version "$CHART_VERSION"
}

release_validate_operator_version() {
  local value="$1"
  local name="${2:-OPERATOR_VERSION}"

  if [[ ! "$value" =~ $OPERATOR_VERSION_REGEX ]]; then
    echo "$name must look like v1.2.3 or v1.2.3-rc1; got $value" >&2
    exit 1
  fi
}

release_validate_chart_version() {
  local value="$1"
  local name="${2:-CHART_VERSION}"

  if [[ ! "$value" =~ $CHART_VERSION_REGEX ]]; then
    echo "$name must look like 1.2.3 or 1.2.3-rc1; got $value" >&2
    exit 1
  fi
}

# release_load_metadata reads the four release facts from handoff.json into the
# shared OPERATOR_VERSION/PREVIOUS_OPERATOR_VERSION/CHART_VERSION/RUNNER_K6_VERSION
# variables, so callers don't each re-jq the same fields.
release_load_metadata() {
  local file="${1:-.github/release/handoff.json}"

  if [[ ! -f "$file" ]]; then
    echo "Release metadata file not found: $file" >&2
    exit 1
  fi

  # These are consumed by the scripts that source this lib, not here.
  # shellcheck disable=SC2034
  OPERATOR_VERSION="$(jq -r '.operator_version' "$file")"
  # shellcheck disable=SC2034
  PREVIOUS_OPERATOR_VERSION="$(jq -r '.previous_operator_version' "$file")"
  # shellcheck disable=SC2034
  CHART_VERSION="$(jq -r '.chart_version' "$file")"
  # shellcheck disable=SC2034
  RUNNER_K6_VERSION="$(jq -r '.runner_k6_version' "$file")"
}

# Canonical contents of the release-managed lines, defined once so prepare
# (which writes them) and validate (which checks them) cannot drift apart.
release_makefile_version_line() {
  printf 'VERSION ?= %s' "${1#v}"
}

release_chart_appversion_line() {
  printf 'appVersion: "%s"' "${1#v}"
}

release_chart_version_line() {
  printf 'version: %s' "$1"
}

release_values_tag_line() {
  printf '    tag: controller-%s' "$1"
}

# release_versioning_row: $1 operator version, $2 runner k6 version.
release_versioning_row() {
  printf '| %s | [runner-%s](ghcr.io/grafana/k6-operator:runner-%s) | %s |' \
    "$1" "$1" "$1" "$2"
}
