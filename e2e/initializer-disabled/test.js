import { Environment } from 'k6/x/environment';
import { sleep } from 'k6';
import { expect } from '../assertions.js';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "./"
const TEST_NAME = "k6-sample"

const env = new Environment({
  name: "initializer-disabled",
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

  err = env.wait({
    kind: "TestRun",
    name: TEST_NAME,
    namespace: "default",
    status_key: "stage",
    status_value: "finished",
  }, {
    timeout: "5m",
    interval: "1m",
  });

  expect(err, "wait returns").toBeNull();

  // Check total pod count
  // With initializer disabled: runner + starter = 2 pods (initializer should be skippd)
  let allPods = env.getN("pods", {
    "namespace": "default",
    "app": "k6",
    "k6_cr": TEST_NAME,
  });

  const expectedPods = 2;
  expect(allPods, "initializer should not exist when disabled").toBe(expectedPods);

  console.log("SUCCESS: No initializer pod created, total pods: " + allPods);
}

export function teardown() {
  console.log("delete returns", env.delete());
}
