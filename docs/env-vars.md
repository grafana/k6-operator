# Environment variables

There are some tricky scenarios that can come up when trying to pass environment variables to k6-operator tests, esp. when archives are involved. Let's describe them.

Assuming I have a `test.js` with the following options:

```js
const VUS = __ENV.TEST_VUS || 10000;

export const options = {
    vus: VUS,
    duration: '300s',
    setupTimeout: '600s'
};
```

Firstly, it is a recommended practice to set the default value of environment variable definition as here:
```js
const VUS = __ENV.TEST_VUS || 10000;
```

This way, the script won't break even if there are changes in the setup.

Within k6-operator context, there are several ways the `TEST_VUS` can be configured for `test.js`. Let's look into some of them.

## ConfigMap with `test.js`

ConfigMap `env-test` contains a `test.js` file. TestRun definition:

```yaml
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: k6-sample
spec:
  parallelism: 2
  script:
    configMap:
      name: "env-test"
      file: "test.js"
  runner:
    env:
      - name: TEST_VUS
        value: "42"
```

In this case, there will be 42 VUs in total, equally split between 2 runners. This can be confirmed with [k6 execution API](https://grafana.com/docs/k6/latest/javascript-api/k6-execution/).

## ConfigMap has `archive.tar` with system env vars

`archive.tar` was created with the following command:

```bash
TEST_VUS=4 k6 archive --include-system-env-vars test.js
```

ConfigMap `env-test` contains `archive.tar`. TestRun definition:

```yaml
apiVersion: k6.io/v1alpha1
kind: TestRun
metadata:
  name: k6-sample
spec:
  parallelism: 2
  script:
    configMap:
      name: "env-test"
      file: "archive.tar"
  runner:
    env:
      - name: TEST_VUS
        value: "42"
```

In this case, there will be 4 VUs in total, split between 2 runners, and at the same time `TEST_VUS` env variable inside the script will be equal to 42.

## `ConfigMap` has `archive.tar` without system env vars

`archive.tar` was created with the following command:

```bash
TEST_VUS=4 k6 archive test.js
```

ConfigMap `env-test` and TestRun `k6-sample` are as in previous case. Then there will be 10000 VUs in total, split between 2 runners. And `TEST_VUS` inside the script will be 42 again.

# Conclusion

This is not fully k6-operator defined behaviour: the operator only passes the env vars to the pods. But if one uses `k6 archive` to create a script, the VUs will be defined at that step. This difference between configuration for `k6 archive` and `k6 run` is described in the docs [here](https://grafana.com/docs/k6/latest/misc/archive/#contents-of-an-archive-file) and [here](https://grafana.com/docs/k6/latest/using-k6/environment-variables/).
