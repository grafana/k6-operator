# E2E tests for k6-operator

This is a basic suite of E2E tests for k6-operator, covering some of the main use cases. It can be executed all at once (sequentially) or by picking up a single test:

```sh
# execute all tests one-by-one
./run-tests.sh

# execute just one test by name of the folder
./run-tests.sh -p basic-testrun-1

# NOTE: `ipv6` folder is currently an exception and cannot be started in this way
```

It is assumed that there is a Kubernetes cluster to execute the tests in and `kubectl` with access to it.

The [vcluster](https://www.vcluster.com/install) CLI should also be installed as it is currently a dependency of xk6-environment. See the [issue](https://github.com/grafana/xk6-environment/issues/1) for details.

## Under the hood

`run-tests.sh` does not build any images but it can be customized with custom image and a tag for k6-operator image. At the same time the script uses the current k6-operator folder to get manifests. So for example, in order to test a certain branch, one has to switch to that branch locally first.

Each test is executed with [xk6-environment](https://github.com/grafana/xk6-environment) extension, bootstraping a virtual cluster and full isolation per test. After a test is finished, everything connected to it is removed, leaving a cluster in its initial state. While the test executes, it is not recommended to interact with the cluster unless it's for monitoring or debugging purposes.

### Build a custom image

As an example, the following commands can be used to set up a custom image of k6-operator with the local [`kind` cluster](https://kind.sigs.k8s.io):

```sh
IMG_NAME=k6operator IMG_TAG=foo make docker-build
kind load docker-image k6operator:foo

# Check where image is on the cluster to be certain of its name:
docker exec -it kind-control-plane crictl images | grep k6operator

# Start a suite with the custom image:
./run-tests.sh  -i docker.io/library/k6operator:foo
```

### GCk6 tests

In order to execute Grafana Cloud k6 tests (cloud output or, in the future, PLZ), one is expected to create environment variables containing the tokens for authentication:

```sh
# personal GCk6 token
# Encode your token with base64:
echo -n '<MY PERSONAL TOKEN HERE>' | base64

# The output can contain an additional newline in terminal so remove it, then export it like this:
export CLOUD_TOKEN=... # in base64!
```

A similar process will be needed for an organization token (required for PLZ):
```sh
echo -n '<MY ORG TOKEN HERE>' | base64

export CLOUD_ORG_TOKEN=... # in base64!
```

If the cloud environment variables are present in a `stack.env` file, they can be quickly exported before running the test suite with this command:
```sh
export $(cat stack.env | xargs)
```

## How to add a test

Firstly, the existing tests can be used as a basis for many additional experiments. Otherwise, the skeleton for the test looks like this:

```sh
new-test
├── manifests
│   ├── configmap.yaml # contains k6 script for the TestRun
│   └── kustomization.yaml # a gathering point for all required manifests
├── test.js # the test with xk6-environment setup
└── testrun.yaml # TestRun to test
```

`kustomization.yaml` file must include the `latest` folder. To create it quickly, this shortcut can be used:
```sh
cd $folder/manifests
kustomize create --autodetect --recursive --resources ../../latest/
```