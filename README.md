 ![data flow](assets/data-flow.png)

# k6 Operator

`grafana/k6-operator` is a Kubernetes operator for running distributed k6 tests in your cluster.

Read also the [complete tutorial](https://k6.io/blog/running-distributed-tests-on-k8s/) to learn more about how to use this project.

## Setup

REMOVED

## Usage

Samples are available in `config/samples` and `e2e/`, both for `TestRun` and for `PrivateLoadZone`.

REMOVED

## Uninstallation

You can remove the all resources created by the operator with bundle:
```bash
curl https://raw.githubusercontent.com/grafana/k6-operator/main/bundle.yaml | kubectl delete -f -
```

Or with `make` command:
```bash
make delete
```

## Developing Locally

### Pre-Requisites

- [operator-sdk](https://sdk.operatorframework.io/docs/installation/)
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/)

### Run Tests

#### Test Setup

- `make test-setup` (only need to run once)

#### Run Unit Tests

- `make test`

#### Run e2e Tests

- [install kind and create a k8s cluster](https://kind.sigs.k8s.io/docs/user/quick-start/) (or create your own dev cluster)
- `make e2e` for kustomize and `make e2e-helm` for helm
- validate tests have been run
- `make e2e-cleanup`

## See also

- [Running distributed k6 tests on Kubernetes](https://k6.io/blog/running-distributed-tests-on-k8s/)
