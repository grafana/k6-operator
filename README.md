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
  script: k6-test
  separate: false
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
  script: crocodile-stress-test
  image: k6-prometheus:latest
  arguments: --out prometheus
  ports:
  - containerPort: 5656
    name: metrics
```

Note that we are replacing the test job image (`k6-prometheus:latest`), passing required arguments to `k6`
(`--out prometheus`), and also exposing the ports required for Prometheus to scrape the metrics
(in this case, that's port `5656`)

## Uninstallation
Running the command below will delete all resources created by the operator.
```bash
$ make delete
```

## See also

- [Running distributed k6 tests on Kubernetes](https://k6.io/blog/running-distributed-tests-on-k8s/)
