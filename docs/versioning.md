## k6 versioning

Since around v0.0.7 k6-operator has been releasing a default runner image together with each pre-release and release. It is built from the `grafana/k6:latest` so the version of k6 in the default runner's image highly depends on the time of the build itself. For now let's keep this info in a table for the historical record and for ease of reference.


| k6-operator version | k6 version in runner image |
|:---------:|:-------:|
| v0.0.7rc  | v0.33.0 |
| v0.0.7rc2 | v0.33.0 |
| v0.0.7rc3 | v0.34.1 |
| v0.0.7rc4 | v0.36.0 |
| v0.0.7rc  | v0.38.2 |
| v0.0.8rc1 | v0.38.3 |
| v0.0.8rc2 |   N/A   |
| v0.0.8rc3 | v0.40.0 |
| v0.0.8    | v0.41.0 |
| v0.0.9rc1 | v0.41.0 |
| v0.0.9rc2 | v0.42.0 |
| v0.0.9rc3 | v0.42.0 |
| v0.0.9    |  |

### What was used before?

Previously k6-operator has been relying on `loadimpact/k6:latest` image by default. That changed with addition of Istio support and then CI additions. Since then k6-operator is using `ghcr.io/grafana/operator:latest-runner` as a default image.