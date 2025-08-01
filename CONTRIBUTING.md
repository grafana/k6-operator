# Contributing guide

The k6 Operator has received a lot of amazing improvements from contributors over the years. Your help is very much welcome!

There are several ways to contribute to the k6 Operator.

## Opening issues

Before opening an issue, make sure to search the existing [issues](https://github.com/grafana/k6-operator/issues), to ensure that it's not a duplicate. Search closed issues as well, in case it was already resolved.

If your issue already exists, consider upvoting it or adding a comment with additional details about your use case: that is also valuable.

### Bug report

If you have encountered a repeatable bug in your usage of the k6 Operator, please open a bug report with [this template](https://github.com/grafana/k6-operator/issues/new?assignees=&labels=bug&projects=&template=bug.yaml).

Make sure to provide detailed instructions and commands on how to repeat the bug on another cluster. A bug must be easily repeatable by anyone. If it's not easily repeatable, consider asking a question on the [community forum](https://community.grafana.com/c/grafana-k6/k6-operator/73) first.

If there is a third-party software involved, provide links to their documentation and instructions to follow. For example, if some tool must be installed to repeat the bug, it's best to include the exact command that installs it in the description. This simplifies the work of a reviewer and, therefore, speeds up the answer.

### Feature request

When writing a new feature request, please use [this template](https://github.com/grafana/k6-operator/issues/new?assignees=&labels=enhancement&projects=&template=feat_req.yaml).

Add a description of your use case that you're trying to solve. The k6 Operator already supports a lot of use cases, and in many scenarios, it is sufficient. Adding a new feature should be done with a clear understanding of why it's needed. The better you describe your use case, the higher the chance that this feature request will be implemented and a corresponding PR will be merged.

## Opening PRs

When submitting a PR, it's preferable to work on an open issue. If an issue does not exist, [create one](#opening-issues). An issue allows us to validate the problem, gather additional feedback from the community, and avoid unnecessary work.

Some GitHub issues have a ["good first issue" label](https://github.com/grafana/k6-operator/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22). These are the issues that should be good for newcomers. The issues with the ["help wanted" label](https://github.com/grafana/k6-operator/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) are the ones that could use some community help or additional user feedback.

All contributors must sign the CLA. If you can't sign the CLA for some reason, sadly, we'll not be able to accept the PR.

If the PR is centered on the k6 Operator itself, read the [next section](#development).

If the PR is Helm-centered, check out the [Helm section](#helm).

## Development

There are many options for setting up a local Kubernetes cluster for development, and any of them can be used for local development of the k6 Operator. One option is to create a [kind cluster](https://kind.sigs.k8s.io/docs/user/quick-start/).

Additionally, you'll need to install the following tooling:

- [Golang](https://go.dev/doc/install)
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/): it can also be quickly installed with `make kustomize`.
- [operator-sdk](https://sdk.operatorframework.io/docs/installation/): optional, as most changes can be done without it.

To build the Docker image, use the `make docker-build` command.

To deploy from the repository, use the `make deploy` command.

An example deployment to a kind cluster can look like this:
```sh
IMG_NAME=k6operator IMG_TAG=foo make docker-build
kind load docker-image k6operator:foo
IMG_NAME=k6operator IMG_TAG=foo make deploy
```

All commands available in the Makefile can be seen with the `make help` command.

There are different additional tools that allow for speeding up the development. For contributors, the development of the k6 Operator is not restricted to any one tool: use whatever works best for you.

As an example, the repo contains the [DevSpace](https://www.devspace.sh/) setup, and it can be used in the following way:

1. Install DevSpace.
1. Deploy k6 Operator with `make deploy`.
1. Run `devspace dev -n k6-operator-system` to start a replacement Pod.
1. Once inside the replacement Pod, run the `go run cmd/main.go` command.
1. Test and debug inside the replacement Pod as needed.
1. Exit the replacement Pod and run `devspace purge` to clean it up.

### Before opening PRs

If there are changes to the Golang code, it makes sense to run the following to preempt the CI errors if any:
```sh
# To execute unit tests.
make test

# To execute Golang linting rules.
make lint
```

If there are changes to the CRDs definitions, make sure that the output of the following commands is included in the PR:
```sh
make manifests
make generate-crd-docs
```

### E2E tests

The k6 Operator repository contains the E2E test suite with the most common scenarios.

As described in the test suite's [Readme](https://github.com/grafana/k6-operator/tree/main/e2e#e2e-tests-for-k6-operator), each test is executed in a virtual cluster to isolate the test and its results. It takes some time to run them all.

The test suite is executed manually at the moment, with the Bash script, but it is preferable to run it before submitting a PR, if possible, and check for errors in the output.

## Helm

Makefile has some commands that can help with the changes to the chart:
```sh
make helm-template

make deploy-helm

make e2e-helm
```

When opening a PR, do not change the version of the chart: this will trigger a release, and releases are to be done by the maintainer.

### Changes to the `values.yaml`

Whenever you make changes to the `values.yaml`, run these two commands:
1. `make helm-docs` to generate a new Readme for the Chart.
1. `make helm-schema` to generate a new `values.schema.json` file. Pay attention to the comments in `values.yaml`: those comments control the JSON schema definition. The comments should be correct and sensible.

The output of both commands must be included in the PR.

### Testing the upgrade

Sometimes it's important to test that the modified Helm chart will work correctly after an upgrade. To do that, run the following:
```sh
helm repo update # to fetch the latest release
helm install k6-operator grafana/k6-operator
```

Check that the latest release of the charts is running.

Next, switch to your GitHub branch locally and upgrade from the local folder:
```sh
helm upgrade k6-operator ./charts/k6-operator/
```

Check if the upgraded chart is working as expected.

If there's a need to pass certain values to the chart, here's a quick shortcut:
```sh
helm upgrade k6-operator ./charts/k6-operator/ --set=manager.image.tag=XXXX
```

## Documentation

The k6 Operator repo contains only internal documentation. The official [user documentation](https://grafana.com/docs/k6/latest/set-up/set-up-distributed-k6/) for the k6 Operator is in the [k6-docs repo](https://github.com/grafana/k6-docs). If you see something incorrect or outdated in it, you are also welcome to open an issue or a PR in the k6-docs.
