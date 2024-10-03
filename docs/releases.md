## Release workflow

Current release process is rather heavy on manual interventions:

1. _manual_ Create a Github release.
2. "Release" workflow is triggered:
- build of new Docker images from `main`
- PR to update bundle
3. _manual_ Review and merge PR with bundle update.
4. _manual_ Commit and push the following changes:
- Update Makefile with latest version.
- Update `docs/versioning.md`.
- Update CRDs in Helm chart if needed.
- Update k6-operator's version in `values.yaml` and bump `Chart.yaml`
- Run `helm-docs` to update the auto-generated documentation for the Chart
- Commit the changes:
    ```bash
    git add charts/k6-operator/Chart.yaml charts/k6-operator/README.md charts/k6-operator/values.yaml docs/versioning.md Makefile
    git commit -m 'release: update for v0.0.x'
    ```
5. "Helm release" workflow is triggered, publishing to Helm Grafana repo.
