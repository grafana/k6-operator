import { Environment } from 'k6/x/environment';
import { sleep } from 'k6';
import { expect } from '../assertions.js';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "./"
const TEST_NAME = "k6-missing-k6"

const env = new Environment({
  name: "initializer-missing-k6-logs",
  implementation: "vcluster",
  initFolder: PARENT + "manifests",
})

export function setup() {
  console.log("init returns", env.init());
  sleep(0.5);
}

export default function () {
  let err = env.apply(PARENT + "testrun.yaml");
  console.log("apply testrun returns", err)
  expect(err, "apply testrun returns").toBeNull();

  err = env.wait({
    kind: "TestRun",
    name: TEST_NAME,
    namespace: "default",
    status_key: "stage",
    status_value: "error",
  }, {
    timeout: "3m",
    interval: "10s",
  });
  expect(err, "testrun should reach error stage").toBeNull();

  err = env.apply(PARENT + "log-checker.yaml");
  console.log("apply log checker returns", err)
  expect(err, "apply log checker returns").toBeNull();

  err = env.wait({
    kind: "Job",
    name: "initializer-log-checker",
    namespace: "default",
    condition_type: "Complete",
    value: "True",
  }, {
    timeout: "3m",
    interval: "10s",
  });
  expect(err, "initializer logs should show missing k6").toBeNull();
}

export function teardown() {
  console.log("delete returns", env.delete());
}
