 ![data flow](assets/data-flow.png)

# k6 Operator

> ### ⚠️ Experimental
>
> This project is **experimental** and changes a lot between commits.
> Use at your own risk. 

`k6io/operator` is a kubernetes operator for running distributed k6 tests in your cluster.

Read also the [complete tutorial](https://k6.io/blog/running-distributed-tests-on-k8s/) to learn more about how to use this project.

## Setup

### Deploying the operator
Install the operator by running the command below:

```bash
$ make deploy
``` 

### Installing the CRD

The k6 operator includes one custom resource called `K6`. This will be automatically installed when you do a
deployment, but in case you want to do it yourself, you may run the command below:

```bash
$ make install
```

## Usage

Two samples are available in `config/samples`, one for a test script and one for an actual test run.

### Adding test scripts

The operator utilises `ConfigMap`s to serve test scripts to the jobs. To upload your own test script, run the following:

```bash
$ kubectl create configmap my-test --from-file /path/to/my/test.js
``` 

### Executing tests
Tests are executed by applying the custom resource `K6` to a cluster where the operator is running. The properties
of a test run are few, but allow you to control some key aspects of a distributed execution.

```yaml
# k6-resource.yml

apiVersion: k6.io/v1alpha1
kind: K6
metadata:
  name: k6-sample
spec:
  parallelism: 4
  script: 
    configMap:
      name: k6-test
      file: test.js
  separate: false
  runner:
    image: <custom-image>
    metadata:
      labels:
        cool-label: foo
      annotations:
        cool-annotation: bar
    resources:
      limits:
        cpu: 200m
        memory: 1000mi
      requests:
        cpu: 100m
        memory: 500Mi
  starter:
    image: <custom-image>
    metadata:
      labels:
        cool-label: foo
      annotations:
        cool-annotation: bar
```

The test configuration is applied using

```bash
$ kubectl apply -f /path/to/your/k6-resource.yml
```
     
#### Parallelism
How many instances of k6 you want to create. Each instance will be assigned an equal execution segment. For instance,
if your test script is configured to run 200 VUs and parallelism is set to 4, as in the example above, the operator will
create four k6 jobs, each running 50 VUs to achieve the desired VU count.

#### Script
The name of the config map that includes our test script. In the example in the [adding test scripts](#adding-test-scripts)
section, this is set to `my-test`.

#### Separate
Toggles whether the jobs created need to be distributed across different nodes. This is useful if you're running a
test with a really high VU count and want to make sure the resources of each node won't become a bottleneck.

#### Serviceaccount

Defines the service account to be used for runners and starter pods. This allows for pulling images from a custom repository.

#### Runner

Defines options for the test runner pods. This includes:

* passing resource limits and requests
* passing in labels and annotations
* passing in affinity and anti-affinity
* passing in a custom image

#### Starter

Defines options for the starter pod. This includes:

* passing in custom image
* passing in labels and annotations



### Cleaning up between test runs
After completing a test run, you need to clean up the test jobs created. This is done by running the following command:
```bash
$ kubectl delete -f /path/to/your/k6-resource.yml
```

### Using extensions
By default, the operator will use `loadimpact/k6:latest` as the container image for the test jobs. If you want to use
extensions built with [xk6](https://github.com/k6io/xk6) you'll need to create your own image and override the `image`
property on the `K6` kubernetes resource. For example, the following Dockerfile can be used to create a container
image using github.com/szkiba/xk6-prometheus as an extension:


```Dockerfile
# Build the k6 binary with the extension
FROM golang:1.16.4-buster as builder

RUN go install github.com/k6io/xk6/cmd/xk6@latest
RUN xk6 build --output /k6 --with github.com/szkiba/xk6-prometheus@latest

# Use the operator's base image and override the k6 binary
FROM loadimpact/k6:latest
COPY --from=builder /k6 /usr/bin/k6
```

If we build and tag this image as `k6-prometheus:latest`, when we can use it as follows:

```yaml
# k6-resource-with-extensions.yml

apiVersion: k6.io/v1alpha1
kind: K6
metadata:
  name: k6-sample-with-extensions
spec:
  parallelism: 4
  script: 
    configMap: 
      name: crocodile-stress-test
      file: test.js
  image: k6-prometheus:latest
  arguments: --out prometheus
  ports:
  - containerPort: 5656
    name: metrics
```

Note that we are replacing the test job image (`k6-prometheus:latest`), passing required arguments to `k6`
(`--out prometheus`), and also exposing the ports required for Prometheus to scrape the metrics
(in this case, that's port `5656`)

If using the Prometheus Operator, you'll also need to create a pod monitor: 

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: k6-monitor
spec:
  selector:
    matchLabels:
      app: k6
  podMetricsEndpoints:
  - port: metrics
```

### Scheduling Tests

While the k6 operator doesn't support scheduling k6 tests directly, the recommended path for scheduling tests is to use the cronjobs object from k8s directly. The cron job should run on a schedule and run a delete and then apply of a k6 object

Running these tests requires a little more setup, the basic steps are:

1. Create a configmap of js test files (Covered above)
1. Create a configmap of the yaml for the k6 job
1. Create a service account that lets k6 objects be created and deleted
1. Create a cron job that deletes and applys the yaml

Add a configMapGenerator to the kustomization.yaml:

```yaml
configMapGenerator:
  - name: <test-name>-config
    files:
      - <test-name>.yaml
```

Then we are going to create a service account for the cron job to use:

This is required to allow the cron job to actually delete and create the k6 objects.

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k6-<namespace>
rules:
  - apiGroups:
      - k6.io
    resources:
      - k6s
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: k6-<namespace>
roleRef:
  kind: ClusterRole
  name: k6-<namespace>
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: k6-<namespace>
    namespace: <namespace>
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k6-<namespace>
```

We're going to create a cron job:

```yaml
# snapshotter.yml
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: <test-name>-cron
spec:
  schedule: "<cron-schedule>"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccount: k6
          containers:
            - name: kubectl
              image: bitnami/kubectl
              volumeMounts:
                - name: k6-yaml
                  mountPath: /tmp/
              command:
                - /bin/bash
              args:
                - -c
                - "kubectl delete -f /tmp/<test-name>.yaml; kubectl apply -f /tmp/<test-name>.yaml"
          restartPolicy: OnFailure
          volumes:
            - name: k6-yaml
              configMap:
                name: <test-name>-config
```


## Uninstallation
Running the command below will delete all resources created by the operator.
```bash
$ make delete
```

## Developing Locally

### Run Tests

#### Pre-Requisites

- [operator-sdk](https://sdk.operatorframework.io/docs/installation/)

#### Test Setup

- `make test-setup` (only need to run once)

#### Run Unit Tests

- `make test`

#### Run e2e Tests

- [install kind and create a k8s cluster](https://kind.sigs.k8s.io/docs/user/quick-start/) (or create your own dev cluster) 
- `make e2e`
- validate tests have been run
- `make e2e-cleanup`

## See also

- [Running distributed k6 tests on Kubernetes](https://k6.io/blog/running-distributed-tests-on-k8s/)
