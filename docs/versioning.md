## k6 versioning

A user can set custom k6 image for runners as an optional field. The k6-operator has a hard-coded default value for k6 image.

Initially k6-operator has been relying on `loadimpact/k6:latest` image by default. That changed with addition of Istio support, in v0.0.7, when k6-operator switched to building `ghcr.io/grafana/operator:latest-runner` images, with `scuttle`, and using them as a default image. This image is built from the `loadimpact/k6:latest` / `grafana/k6:latest` during k6-operator release. So the version of k6 in it highly depends on the time of the release itself. We keep this version mapping in a table below for the historical record and for ease of reference. The tag `latest-runner` points to the last row in the table.

Since v0.0.23 ([issue](https://github.com/grafana/k6-operator/issues/452)), k6-operator uses `grafana/k6:latest` as a default image. But we continue to build runners with `scuttle` until Istio support is removed ([issue](https://github.com/grafana/k6-operator/issues/195)).

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
| v0.0.19             | [runner-v0.0.19](ghcr.io/grafana/k6-operator:runner-v0.0.19)           | v0.56.0 |
| v0.0.20             | [runner-v0.0.20](ghcr.io/grafana/k6-operator:runner-v0.0.20)           | v1.0.0-rc1 |
| v0.0.21             | [runner-v0.0.21](ghcr.io/grafana/k6-operator:runner-v0.0.21)           | v1.0.0 |
| v0.0.22             | [runner-v0.0.22](ghcr.io/grafana/k6-operator:runner-v0.0.22)           | v1.1.0 |
| v0.0.23             | [runner-v0.0.23](ghcr.io/grafana/k6-operator:runner-v0.0.23)           | v1.1.0 |
| v1.0.0              | [runner-v1.0.0](ghcr.io/grafana/k6-operator:runner-v1.0.0)            | v1.2.3 |
| v1.1.0              | [runner-v1.1.0](ghcr.io/grafana/k6-operator:runner-v1.1.0)            | v1.4.0 |
| v1.1.1              | [runner-v1.1.1](ghcr.io/grafana/k6-operator:runner-v1.1.1)            | v1.4.2 |
| v1.2.0              | [runner-v1.2.0](ghcr.io/grafana/k6-operator:runner-v1.2.0)            | v1.5.0 |
| v1.3.0              | [runner-v1.3.0](ghcr.io/grafana/k6-operator:runner-v1.3.0)            | v1.6.1 |
| v1.3.1              | [runner-v1.3.1](ghcr.io/grafana/k6-operator:runner-v1.3.1)            | v1.7.0 |
| v1.3.2              | [runner-v1.3.2](ghcr.io/grafana/k6-operator:runner-v1.3.2)            | v1.7.0 |
| v1.4.0              | [runner-v1.4.0](ghcr.io/grafana/k6-operator:runner-v1.4.0)            | v2.0.0-rc1 |
