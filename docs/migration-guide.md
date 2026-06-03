# Migrating from single instance to multiple instances

This guide describes how to migrate an existing k6 setup from a single runner to multiple runners using k6-operator. It also explains the relationship between k6 executors and execution segments, which is important for understanding how distributed runs work.

## How k6-operator distributes a test

When you set `parallelism: N` in a `TestRun`, k6-operator starts `N` runner pods. Each runner receives an [execution segment](https://grafana.com/docs/k6/latest/using-k6/k6-options/reference/#execution-segment) that limits it to a portion of the total work. For example, with `parallelism: 2`, runner 0 gets segment `0:1/2` and runner 1 gets segment `1/2:1`.

This means all runners execute the **same script** but each covers a different slice of the virtual users and iterations. The total load across all runners equals what you define in the script options.

## Single instance setup

A typical single-instance run:

```js
export const options = {
  vus: 50,
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

This starts 50 VUs from a single pod running for 1 minute.

## Migrating to multiple instances

To spread the same load across multiple runners, increase `parallelism` and scale up VUs proportionally. The formula is simple:

> **total VUs = original VUs per instance * number of instances**

So if you were running 1 instance with 50 VUs and you want to scale to 4 instances:

```js
export const options = {
  vus: 200,   // 50 * 4
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

Each runner handles 50 VUs, for a total of 200 VUs across the cluster. This replicates the original load profile on each node.

## Working with other executors

The same principle applies to other [k6 executors](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/).

### constant-arrival-rate

```js
export const options = {
  scenarios: {
    contacts: {
      executor: 'constant-arrival-rate',
      rate: 400,          // 100 * 4 runners
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 80,
      maxVUs: 400,
    },
  },
};
```

With `parallelism: 4`, each runner fires 100 requests per second, for 400 req/s total.

### ramping-vus

```js
export const options = {
  stages: [
    { target: 400, duration: '30s' },  // ramps up to 400 total (100 per runner)
    { target: 0,   duration: '30s' },
  ],
};
```

k6 splits the VU ramp proportionally between runners using execution segments, so the total ramp shape is preserved.

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

| Original setup | Multi-instance equivalent |
|---|---|
| `parallelism: 1`, `vus: Y` | `parallelism: X`, `vus: X * Y` |
| `parallelism: 1`, `rate: R` | `parallelism: X`, `rate: X * R` |
| `parallelism: 1`, stages ramp to `T` | `parallelism: X`, stages ramp to `X * T` |

The key rule: multiply the VU or rate count in your script by the number of runners you plan to use. k6-operator takes care of the rest by splitting execution segments evenly across runners.
