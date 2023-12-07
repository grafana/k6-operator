## Release workflow

Current release process is rather heavy on manual interventions:

1. _manual_ Create a Github release.
2. "Release" workflow is triggered:
- build of new Docker images from `main`
- PR to update bundle
3. _manual_ Review and merge PR with bundle update.
4. _manual_ Commit and push the following changes:
- Update Makefile with latest version, as well as `docs/versioning.md`
- Update k6-operator's version in `values.yaml` and bump `Chart.yaml`
- Run `helm-docs` to update the auto-generated documentation for the Chart
5. "Helm release" workflow is triggered, publishing to Helm Grafana repo.
