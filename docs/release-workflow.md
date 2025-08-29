## Release workflow

The current release process is rather heavy on manual interventions:

1. _manual_ Create a Github release.
2. "Release" workflow is triggered:
- Build new Docker images from `main`
- PR to update bundle
3. _manual_ Review and merge PR with bundle update.
4. _manual_ Commit and push the following changes:
- Update Makefile with latest version.
- Update `docs/versioning.md`.
- Run `make generate-crd-docs` to update the auto-generated documentation for the CRDs.
- Update CRDs in Helm chart if needed.
- Update k6-operator's version in `values.yaml` and bump `Chart.yaml`
- Run `make helm-docs` to update the auto-generated documentation for the Chart
- Run `make helm-schema` to update the schema file.
- Commit the changes:
    ```bash
    git add charts/k6-operator/Chart.yaml charts/k6-operator/README.md charts/k6-operator/values.yaml charts/k6-operator/values.schema.json docs/versioning.md Makefile docs/crd-generated.md
    git commit -m 'release: update for v0.0.x'
    ```
5. "Helm release" workflow is triggered, publishing to Helm Grafana repo.

### Errors on release

Currently, if the Helm JSON schema is not up-to-date, the Helm release will fail. [`helm-release.yaml` workflow](https://github.com/grafana/k6-operator/blob/main/.github/workflows/helm-release.yaml) will create a PR with the necessary changes and `exit 1` to force the maintainer to review the changes and re-run the release workflow.