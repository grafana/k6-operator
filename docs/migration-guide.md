# Migrating from single instance to multiple instances

This guide describes how to migrate an existing k6 setup from a single runner to multiple runners using k6-operator. It also explains the relationship between k6 executors and execution segments, which is important for understanding how distributed runs work.

## How k6-operator distributes a test

When you set `parallelism: N` in a `TestRun`, k6-operator starts `N` runner pods. Each runner receives an [execution segment](https://grafana.com/docs/k6/latest/using-k6/k6-options/reference/#execution-segment) that limits it to a portion of the total work. For example, with `parallelism: 2`, runner 0 gets segment `0:1/2` and runner 1 gets segment `1/2:1`.

This means all runners execute the **same script** but each covers a different slice of the virtual users and iterations. The total load across all runners equals what you define in the script options.

## Single instance setup

A typical single-instance run drives all virtual users from one pod:

```js
export const options = {
  vus: 200,
  duration: '1m',
};
```

And a `TestRun` with parallelism 1:

```yaml
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: single-instance
spec:
  parallelism: 1
  script:
    configMap:
      name: k6-test
      file: test.js
```

This starts 200 VUs from a single pod running for 1 minute.

## Migrating to multiple instances

The usual reason to move to multiple runners is to spread an existing load across more pods, for example when a single pod no longer has enough CPU or memory to drive all the VUs. With k6-operator you keep the same values in your script options and only increase `parallelism`. The operator divides the total load across the runners for you using execution segments, so you do not multiply anything yourself.

> **The values in your script options are always the total load. `parallelism` only controls how many runner pods share that total.**

So to move the 200 VU test above onto 4 runners, leave the script unchanged and set `parallelism: 4`:

```js
export const options = {
  vus: 200,   // unchanged: this is the total across all runners
  duration: '1m',
};
```

```yaml
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: multi-instance
spec:
  parallelism: 4
  script:
    configMap:
      name: k6-test
      file: test.js
```

Each runner pod now handles about 50 VUs (200 / 4), for the same total of 200 VUs across the cluster. The load profile is unchanged, it is just distributed across pods.

### Increasing total load at the same time

If you also want to drive more load than before, raise the values in your script options. For example, setting `vus: 400` with `parallelism: 4` runs about 100 VUs per runner for 400 VUs total. Changing the total load and changing `parallelism` are independent decisions: `parallelism` distributes whatever total you configure.

## Working with other executors

The same rule applies to other [k6 executors](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/): the values you set are the total, and k6-operator splits them across runners.

### constant-arrival-rate

```js
export const options = {
  scenarios: {
    contacts: {
      executor: 'constant-arrival-rate',
      rate: 400,          // total request rate across all runners
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 80,
      maxVUs: 400,
    },
  },
};
```

With `parallelism: 4`, each runner fires about 100 requests per second, for 400 req/s total.

### ramping-vus

```js
export const options = {
  stages: [
    { target: 400, duration: '30s' },  // total target across all runners
    { target: 0,   duration: '30s' },
  ],
};
```

k6 splits the VU ramp proportionally between runners using execution segments, so each runner ramps toward about 100 VUs while the total ramp shape is preserved.

## What stays the same

You do not need to change anything about how your script interacts with the target system or how you handle thresholds. k6 applies [thresholds](https://grafana.com/docs/k6/latest/using-k6/thresholds/) globally: each runner contributes its metrics and the check passes or fails the same way as in a single-instance run.

The `setup()` and `teardown()` functions run **once** per test run, on the initializer pod, not once per runner. This matches single-instance behaviour.

## Checking actual VU distribution

You can verify the distribution using the [k6 execution API](https://grafana.com/docs/k6/latest/javascript-api/k6-execution/):

```js
import exec from 'k6/execution';

export default function () {
  console.log(
    `instance ${exec.instance.vusActive} VUs, segment ${exec.test.options.executionSegment}`
  );
}
```

## Summary

To distribute an existing test across more runners, keep your script options the same and increase `parallelism`:

| Goal | Original setup | Multi-instance setup |
|---|---|---|
| Same total load, distributed | `parallelism: 1`, `vus: Y` | `parallelism: X`, `vus: Y` (unchanged) |
| Same total request rate, distributed | `parallelism: 1`, `rate: R` | `parallelism: X`, `rate: R` (unchanged) |
| Same total ramp, distributed | `parallelism: 1`, stages ramp to `T` | `parallelism: X`, stages ramp to `T` (unchanged) |

The key rule: the VU or rate values in your script are always the total. To distribute an existing test across more runner pods, keep those values the same and increase `parallelism`. Only change the values when you actually want to change the total load. k6-operator takes care of the split by assigning each runner an execution segment.
