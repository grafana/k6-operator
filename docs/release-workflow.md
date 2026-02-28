## Release workflow

The current release process is rather heavy on manual interventions:

1. _manual_ Create a Github release.
2. "Release" workflow is triggered:
    - Builds new Docker images from `main`
    - PR to update bundle
3. _manual_ Review and merge PR with bundle update.
4. _manual_ Commit and push the following changes:
- Update Makefile with latest version.
- Update `docs/versioning.md`.
- Update k6-operator's version in `values.yaml` and bump `Chart.yaml`
- Run the following:
    ```sh
    make generate-crd-docs && make patch-helm-crd && make helm-docs && make helm-schema && make e2e-update-latest
    ```
    - Explanation:
        - `make generate-crd-docs` is to update the auto-generated documentation for the CRDs.
        - `make patch-helm-crd` is to update CRDs in the Helm chart.
        - `make helm-docs` is to update the auto-generated documentation for the Chart.
        - `make helm-schema` is to update the JSON schema file.
        - `make e2e-update-latest` is to update `e2e/latest` folder with the new `bundle.yaml` contents.
    - If any of the above Makefile commands fails, dig into it.
- Commit the changes:
    ```bash
    git add charts/k6-operator/Chart.yaml charts/k6-operator/README.md charts/k6-operator/values.yaml charts/k6-operator/values.schema.json docs/versioning.md Makefile docs/crd-generated.md e2e/latest/*
    git commit -m 'release: update for v1.x.0'
    ```
5. "Helm release" workflow is triggered, publishing to Helm Grafana repo.

### Errors on release

Currently, if the Helm JSON schema is not up-to-date, the Helm release will fail. [`helm-release.yaml` workflow](https://github.com/grafana/k6-operator/blob/main/.github/workflows/helm-release.yaml) will create a PR with the necessary changes and `exit 1` to force the maintainer to review the changes and re-run the release workflow.