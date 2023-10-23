## Release workflow

Current release process is rather heavy on manual interventions:

1. _manual_ Create a Github release.
2. "Release" workflow is triggered:
- build of new Docker images from `main`
- PR to update bundle
3. _manual_ Review and merge PR with bundle update.
4. _manual_ Update operator's version in `values.yaml` and bump `Chart.yaml`.
5. "Helm release" workflow is triggered, publishing to Helm Grafana repo.
6. _manual_ Update Makefile with latest version, as well as `docs/versioning.md`.
