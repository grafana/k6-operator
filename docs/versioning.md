## k6 versioning

Since around v0.0.7 k6-operator has been releasing a default runner image together with each pre-release and release. It is built from the `grafana/k6:latest` so the version of k6 in the default runner's image highly depends on the time of the build itself. For now let's keep this info in a table for the historical record and for ease of reference.

| k6-operator version | runner tag | k6 version in runner image |
|:-------------------:|:----------:|:--------------------------:|
| v0.0.7rc            | [runner-v0.0.7rc](ghcr.io/grafana/operator:runner-v0.0.7rc)   | v0.33.0 |
| v0.0.7rc2           | [runner-v0.0.7rc2](ghcr.io/grafana/operator:runner-v0.0.7rc2) | v0.33.0 |
| v0.0.7rc3           | [runner-v0.0.7rc3](ghcr.io/grafana/operator:runner-v0.0.7rc3) | v0.34.1 |
| v0.0.7rc4           | [runner-v0.0.7rc4](ghcr.io/grafana/operator:runner-v0.0.7rc4) | v0.36.0 |
| v0.0.7              | [runner-v0.0.7](ghcr.io/grafana/operator:runner-v0.0.7)       | v0.38.2 |
| v0.0.8rc1           | [runner-v0.0.8rc1](ghcr.io/grafana/operator:runner-v0.0.8rc1) | v0.38.3 |
| v0.0.8rc2           | [runner-v0.0.8rc2](ghcr.io/grafana/operator:runner-v0.0.8rc2) |   N/A   |
| v0.0.8rc3           | [runner-v0.0.8rc3](ghcr.io/grafana/operator:runner-v0.0.8rc3) | v0.40.0 |
| v0.0.8              | [runner-v0.0.8](ghcr.io/grafana/operator:runner-v0.0.8)       | v0.41.0 |
| v0.0.9rc1           | [runner-v0.0.9rc1](ghcr.io/grafana/operator:runner-v0.0.9rc1) | v0.41.0 |
| v0.0.9rc2           | [runner-v0.0.9rc2](ghcr.io/grafana/operator:runner-v0.0.9rc2) | v0.42.0 |
| v0.0.9rc3           | [runner-v0.0.9rc3](ghcr.io/grafana/operator:runner-v0.0.9rc3) | v0.42.0 |
| v0.0.9              | [runner-v0.0.9](ghcr.io/grafana/operator:runner-v0.0.9)       | v0.44.0 |
| v0.0.10rc1          | [runner-v0.0.10rc1](ghcr.io/grafana/operator:runner-v0.0.10rc1)     | v0.44.1 |
| v0.0.10rc2          | [runner-v0.0.10rc2](ghcr.io/grafana/operator:runner-v0.0.10rc2)     | v0.45.0 |
| v0.0.10rc3          | [runner-v0.0.10rc3](ghcr.io/grafana/k6-operator:runner-v0.0.10rc3)  | v0.45.0 |
| v0.0.10             | [runner-v0.0.10](ghcr.io/grafana/k6-operator:runner-v0.0.10)        | v0.46.0 |
| v0.0.11rc1          | [runner-v0.0.11rc1](ghcr.io/grafana/k6-operator:runner-v0.0.11rc1)     | v0.46.0 |
| v0.0.11rc2          | [runner-v0.0.11rc2](ghcr.io/grafana/k6-operator:runner-v0.0.11rc2)     | v0.46.0 |
| v0.0.11rc3          | [runner-v0.0.11rc3](ghcr.io/grafana/k6-operator:runner-v0.0.11rc3)     | v0.46.0 |
| v0.0.11             | [runner-v0.0.11](ghcr.io/grafana/k6-operator:runner-v0.0.11)           | v0.47.0 |
| v0.0.12rc1          | [runner-v0.0.12rc1](ghcr.io/grafana/k6-operator:runner-v0.0.12rc1)     | v0.47.0 |
| v0.0.12             | [runner-v0.0.12](ghcr.io/grafana/k6-operator:runner-v0.0.12)           | v0.48.0 |
| v0.0.13rc1          | [runner-v0.0.13rc1](ghcr.io/grafana/k6-operator:runner-v0.0.13rc1)     | v0.48.0 |
| v0.0.13             | [runner-v0.0.13](ghcr.io/grafana/k6-operator:runner-v0.0.13)           | v0.49.0 |
| v0.0.14             | [runner-v0.0.14](ghcr.io/grafana/k6-operator:runner-v0.0.14)           | v0.50.0 |
| v0.0.15             | [runner-v0.0.15](ghcr.io/grafana/k6-operator:runner-v0.0.15)           | v0.52.0 |
| v0.0.16             | [runner-v0.0.16](ghcr.io/grafana/k6-operator:runner-v0.0.16)           | v0.52.0 |
| v0.0.17             | [runner-v0.0.17](ghcr.io/grafana/k6-operator:runner-v0.0.17)           | v0.54.0 |
| v0.0.18             | [runner-v0.0.18](ghcr.io/grafana/k6-operator:runner-v0.0.18)           | v0.55.0 |

### What was used before?

Previously k6-operator has been relying on `loadimpact/k6:latest` image by default. That changed with addition of Istio support and then CI additions. Since then k6-operator is using `ghcr.io/grafana/operator:latest-runner` as a default image.