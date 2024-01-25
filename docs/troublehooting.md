# Troubleshooting

Just as any Kubernetes application, k6-operator can get into error scenarios which are sometimes a result of a misconfigured test or setup. This document is meant to help troubleshoot such scenarios quicker.

## Common tricks

### Preparation

> [!IMPORTANT] 
> Before trying to run a script with k6-operator, be it via `TestRun` or via `PrivateLoadZone`, always run it locally:
> 
> ```bash
> k6 run script.js
> ```

If there are going to be environment variables or CLI options, pass them in as well:
```bash
MY_ENV_VAR=foo k6 run script.js --tag my_tag=bar
```

This ensures that the script has correct syntax and can be parsed with k6 in the first place. Additionally, running locally will make it obvious if the configured options are doing what is expected. If there are any errors or unexpected results in the output of `k6 run`, make sure to fix those prior to deploying the script elsewhere.

### `TestRun` deployment

#### The pods

In case of one `TestRun` Custom Resource (CR) creation with `parallelism: n`, there are certain repeating patterns:

1. There will be `n + 2` Jobs (with corresponding Pods) created: initializer, starter, `n` runners.
1. If any of these Jobs did not result in a Pod being deployed, there must be an issue with that Job. Some commands that can help here:
    ```bash
    kubectl get jobs -A
    kubectl describe job mytest-initializer
    ```
1. If one of the Pods was deployed but finished with `Error`, it makes sense to check its logs:
    ```bash
    kubectl logs mytest-initializer-xxxxx
    ```

If the Pods seem to be working but not producing an expected result and there's not enough information in the logs of the Pods, it might make sense to turn on k6 [verbose option](https://grafana.com/docs/k6/latest/using-k6/k6-options/#options) in `TestRun` spec:

```yaml
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: k6-sample
spec:
  parallelism: 2
  script:
    configMap:
      name: "test"
      file: "test.js"
  arguments: --verbose
```

#### k6-operator

Another source of info is k6-operator itself. It is deployed as a Kubernetes `Deployment`, with `replicas: 1` by default, and its logs together with observations about the Pods from [previous subsection](#the-pods) usually contain enough information to glean correct diagnosis. With the standard deployment, the logs of k6-operator can be checked with:

```bash
kubectl -n k6-operator-system -c manager logs k6-operator-controller-manager-xxxxxxxx-xxxxx
```

#### Inspect `TestRun` resource

One `TestRun` CR is deployed, it can be inspected the same way as any other resource:

```bash
kubectl describe testrun my-testrun
```

Firstly, check if the spec is as expected. Then, see the current status:

```yaml
Status:
  Conditions:
    Last Transition Time:  2024-01-17T10:30:01Z
    Message:               
    Reason:                CloudTestRunFalse
    Status:                False
    Type:                  CloudTestRun
    Last Transition Time:  2024-01-17T10:29:58Z
    Message:               
    Reason:                TestRunPreparation
    Status:                Unknown
    Type:                  TestRunRunning
    Last Transition Time:  2024-01-17T10:29:58Z
    Message:               
    Reason:                CloudTestRunAbortedFalse
    Status:                False
    Type:                  CloudTestRunAborted
    Last Transition Time:  2024-01-17T10:29:58Z
    Message:               
    Reason:                CloudPLZTestRunFalse
    Status:                False
    Type:                  CloudPLZTestRun
  Stage:                   error
```

If `Stage` is equal to `error` then it definitely makes sense to check the logs of k6-operator.

Conditions can be used as a source of info as well, but it is a more advanced troubleshooting option that should be used if previous suggestions are insufficient. Note, that conditions that start with `Cloud` prefix matter only in the setting of k6 Cloud test runs, i.e. cloud output and PLZ test runs.

### `PrivateLoadZone` deployment

If `PrivateLoadZone` CR was successfully created in Kubernetes, it should become visible in your account in Grafana Cloud k6 (GCk6) interface soon afterwards. If it doesn't appear in the UI, then there is likely a problem to troubleshoot.

Firstly, go over the [guide](https://grafana.com/docs/grafana-cloud/k6/author-run/private-load-zone-v2/) to double-check if all the steps have been done correctly and successfully.

Unlike `TestRun` deployment, when `PrivateLoadZone` is first created, there are no additional resources deployed. So the only source for troubleshooting are the logs of k6-operator. See the [above subsection](#k6-operator) on how to access its logs. Any errors there might be a hint to what is wrong. See [below](#privateloadzone-subscription-error) for some potential errors explained in more detail.

### Running tests in `PrivateLoadZone`

Each time a user runs a test in a PLZ, for example with `k6 cloud script.js`, there is a corresponding `TestRun` being deployed by k6-operator. This `TestRun` will be deployed in the same namespace as its `PrivateLoadZone`. If such test  is misbehaving (errors out, does not produce expected result, etc.), then one should check:
1) if there are any messages in GCk6 UI
2) if there are any messages in the output of `k6 cloud` command
3) the resources and their logs, the same way as with [standalone `TestRun` deployment](#testrun-deployment)

## Common scenarios

### Where are my env vars...

Some tricky cases with environment variables are described in [this doc](./env-vars.md).

### Tags are not working?!

Currently, tags are a rather common source of frustration in usage of k6-operator. For example:

```yaml
  arguments: --tag product_id="Test A"
  # or
  arguments: --tag foo=\"bar\"
```

Passing the above leads to parsing errors which can be seen in the logs of either initializer or runner Pod, e.g.:
```bash
time="2024-01-11T11:11:27Z" level=error msg="invalid argument \"product_id=\\\"Test\" for \"--tag\" flag: parse error on line 1, column 12: bare \" in non-quoted-field"
```

This is a standard problem with escaping the characters, and there's even an [issue](https://github.com/grafana/k6-operator/issues/211) that can be upvoted.

### Initializer logs an error but it's not about tags

Often, this happens because of lack of attention to the [preparation](#preparation) step. One more command that can be tried here is to run the following:

```bash
k6 inspect --execution-requirements script.js
```

This command is a shortened version of what initializer Pod is executing. If the above command produces an error, it is definitely a problem with the script and should be first solved outside of k6-operator. The error itself may contain a hint to what is wrong, for instance a syntax error.

If standalone `k6 inspect --execution-requirements` executes successfully, then it's likely a problem with `TestRun` deployment specific to your Kubernetes setup. Recommendations here:
- read carefully the output in initializer Pod: is it logged by k6 process or by something else?
  - :information_source: k6-operator expects initializer logs to contain only the output of `k6 inspect`. If there's any other log line present, then k6-operator will fail to parse it and the test will not start. ([issue](https://github.com/grafana/k6-operator/issues/193))
- check events in initializer Job and Pod as they may contain another hint about what is wrong

### Non-existent ServiceAccount

ServiceAccount can be defined as `serviceAccountName` and `runner.serviceAccountName` in PrivateLoadZone and TestRun CRD respectfully. If the specified ServiceAccount does not exist, k6-operator will successfully create Jobs but corresponding Pods will fail to be deployed, and k6-operator will wait indefinitely for Pods to be `Ready`. This error can be best seen in the events of the Job:

```bash
kubectl describe job plz-test-xxxxxx-initializer
...
Events:
  Warning  FailedCreate  57s (x4 over 2m7s)  job-controller  Error creating: pods "plz-test-xxxxxx-initializer-" is forbidden: error looking up service account plz-ns/plz-sa: serviceaccount "plz-sa" not found
```

Currently, k6-operator does not try to analyze such scenarios on its own but we have an [issue](https://github.com/grafana/k6-operator/issues/260) for improvement.

How to fix: incorrect `serviceAccountName` must be corrected and TestRun or PrivateLoadZone resource must be re-deployed.

### Non-existent `nodeSelector`

`nodeSelector` can be defined as `nodeSelector` and `runner.nodeSelector` in PrivateLoadZone and TestRun CRD respectfully. 

This case is very similar to [ServiceAccount one](#non-existent-serviceaccount): the Pod creation will fail, only the error would be somewhat different:

```bash
kubectl describe pod plz-test-xxxxxx-initializer-xxxxx
...
Events:
  Warning  FailedScheduling  48s (x5 over 4m6s)  default-scheduler  0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector.
```

How to fix: incorrect `nodeSelector` must be corrected and TestRun or PrivateLoadZone resource must be re-deployed.

### Insufficient resources

A related problem can happen when the cluster does not have sufficient resources to deploy the runners. There is a higher probability of hitting this issue when setting small CPU and memory limits for runners or using options like `nodeSelector`, `runner.affinity` or `runner.topologySpreadConstraints`, and not having a set of nodes matching the spec. Alternatively, it can happen if there is a high number of runners required for the test (via `parallelism` in TestRun or during PLZ test run) and autoscaling of the cluster has limits on maximum number of nodes and cannot provide the required resources on time or at all.

This case is somewhat similar to the previous two: the k6-operator will wait indefinitely and can be monitored with events in Jobs and Pods. If it is possible to fix the issue with insufficient resources on-the-fly, e.g. by adding more nodes, k6-operator will attempt to continue executing a test run.

### OOM of a runner Pod

If there's at least one runner Pod that OOM-ed, the whole test will be [stuck](https://github.com/grafana/k6-operator/issues/251) and will have to be deleted manually:

```bash
kubectl -f my-test.yaml delete
# or
kubectl delete testrun my-test
``` 

In case of OOM, it makes sense to review k6 script to understand what kind of resource usage this script requires. It may be that the k6 script can be improved to be more performant. Then, set `spec.runner.resources` in TestRun CRD or `spec.resources` in PrivateLoadZone CRD accordingly.

### PrivateLoadZone: subscription error

If there's something off with your k6 Cloud subscription, there will be a 400 error in the logs with the message detailing the problem. For example:

```bash
"Received error `(400) You have reached the maximum Number of private load zones your organization is allowed to have. Please contact support if you want to create more.`. Message from server ``"
```

The most likely course of action in this case is either to check your organization settings in GCk6 or to contact k6 Cloud support.

### PrivateLoadZone: wrong token

There can be two major problems with the token.

1. If token was not created or was created in a wrong location, there will be the following in the logs:
  ```bash
  Failed to load k6 Cloud token	{"namespace": "plz-ns", "name": "my-plz", "reconcileID": "67c8bc73-f45b-4c7f-a9ad-4fd0ffb4d5f6", "name": "token-with-wrong-name", "secretNamespace": "plz-ns", "error": "Secret \"token-with-wrong-name\" not found"}
  ```

2. If token contains a corrupted value or it's not an organizational token, there will be the following error in the logs:
  ```bash
  "Received error `(403) Authentication token incorrect or expired`. Message from server ``"
  ```

### PrivateLoadZone: networking setup

If you see any dial or connection errors in the logs of k6-operator, it makes sense to double-check the networking setup. For PrivateLoadZone to operate, outbound traffic to k6 Cloud [must be allowed](https://grafana.com/docs/grafana-cloud/k6/author-run/private-load-zone-v2/#before-you-begin). The basic way to check the reachability of k6 Cloud endpoints:

```bash
kubectl apply -f https://k8s.io/examples/admin/dns/dnsutils.yaml
kubectl exec -it dnsutils -- nslookup ingest.k6.io
kubectl exec -it dnsutils -- nslookup api.k6.io
```

For more resources on troubleshooting networking, see Kubernetes [official docs](https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/).

### PrivateLoadZone: insufficient resources

The problem is similar to [insufficient resources in general case](#insufficient-resources). But when running a PrivateLoadZone test, k6-operator will wait only for a timeout period (10 minutes at the moment). When the timeout period is up, the test will be aborted by k6 Cloud and marked as such both in PrivateLoadZone and in GCk6. In other words, there is a time limit to fix this issue without restarting the test run.