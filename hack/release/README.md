# Release Scripts

- `check-prepare-preconditions.sh`: Validates release workflow inputs and fails early if the release tag or preparation branch already exists.
- `finalize-release-preflight.sh`: Resolves and validates a merged release preparation PR before finalization creates release side effects.
- `open-release-pr.sh`: Commits generated release artifacts to a release preparation branch and opens the labeled pull request.
- `prepare-release.sh`: Updates release version files, versioning documentation, and generated release metadata.
- `promote-latest-tags.sh`: Promotes versioned controller, runner, and starter image digests to their `latest*` tags.
- `release-lib.sh`: Provides shared release constants, validation helpers, output helpers, and canonical generated-line helpers.
- `render-release-template.sh`: Renders draft GitHub release notes from the release template and latest release metadata.
- `validate-release.sh`: Checks that committed release metadata and generated release files agree before finalization.
