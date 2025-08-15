 ![data flow](assets/data-flow.png)

# k6 Operator

`grafana/k6-operator` is a Kubernetes operator for running distributed [k6](https://github.com/grafana/k6) tests in your cluster. k6 Operator introduces two CRDs:

- `TestRun` CRD
- `PrivateLoadZone` CRD

The `TestRun` CRD is a representation of a single k6 test executed once. `TestRun` supports various configuration options that allow you to adapt to different Kubernetes setups. You can find a description of the more common options [here](https://grafana.com/docs/k6/latest/set-up/set-up-distributed-k6/usage/configure-testrun-crd/), and the full list of options can be found in [docs/crd-generated.md](https://github.com/grafana/k6-operator/blob/main/docs/crd-generated.md).

The `PrivateLoadZone` CRD is a representation of a [load zone](https://grafana.com/docs/grafana-cloud/testing/k6/author-run/use-load-zones/), which is a k6 term for a set of nodes within a cluster designated to execute k6 test runs. `PrivateLoadZone` is integrated with [Grafana Cloud k6](https://grafana.com/products/cloud/k6/) and requires a [Grafana Cloud account](https://grafana.com/auth/sign-up/create-user). You can find a guide describing how to set up a `PrivateLoadZone` [here](https://grafana.com/docs/grafana-cloud/testing/k6/author-run/private-load-zone-v2/), while billing details can be found [here](https://grafana.com/docs/grafana-cloud/cost-management-and-billing/understand-your-invoice/k6-invoice/).

## Installation

Refer to [Install k6 Operator](https://grafana.com/docs/k6/latest/set-up/set-up-distributed-k6/install-k6-operator/) for installation instructions.

## Documentation

You can find the latest k6 Operator documentation in the [Grafana k6 OSS docs](https://grafana.com/docs/k6/latest/set-up/set-up-distributed-k6/).

For additional resources:

- :book: A guide [Running distributed tests](https://grafana.com/docs/k6/latest/testing-guides/running-distributed-tests/).
- :movie_camera: Grafana Office Hours [Load Testing on Kubernetes with k6 Private Load Zones](https://www.youtube.com/watch?v=RXLavQT58YA).

Common samples are available in the `config/samples` and `e2e/` folders in this repo, both for the `TestRun` and `PrivateLoadZone` CRDs.

## Contributing

We are always interested in your feedback! If you encounter problems during the k6 Operator usage, check out the [troubleshooting guide](https://grafana.com/docs/k6/latest/set-up/set-up-distributed-k6/troubleshooting/). If you have questions on how to use the k6 Operator, you can post them on the [Grafana community forum](https://community.grafana.com/c/grafana-k6/k6-operator/73).

Regarding opening issues and development of the k6-operator, please see the [contributing guide](https://github.com/grafana/k6-operator/blob/main/CONTRIBUTING.md).
