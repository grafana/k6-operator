import { Environment } from 'k6/x/environment';
import { sleep } from 'k6';
import { expect } from '../assertions.js';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "./"
const TEST_NAME = "k6-sample-args"
const NAMESPACE = "k6-tests"

const env = new Environment({
  name: "testrun-cloud-output-args",
  implementation: "vcluster",
  initFolder: PARENT + "manifests", // initial folder with everything that wil be loaded at init
})

export function setup() {
  console.log("init returns", env.init());
  // it is best to have a bit of delay between creating a CRD and
  // a corresponding CR, so as to avoid the "no matches" error
  sleep(0.5);
}

// Same scenario as testrun-cloud-output, with the k6 arguments expressed as
// `spec.args`: exact argv elements, including values with spaces and quotes.
//
// The script asserts the `-e` value it receives in its init context, so if the
// arguments do not reach k6 as authored, the initializer and the runners fail
// and the TestRun never reaches `started`. The mismatch itself is reported in
// the logs of the initializer pod.
export default function () {
  let err = env.apply(PARENT + "testrun.yaml");
  console.log("apply testrun returns", err);

  err = env.wait({
    kind: "TestRun",
    name: TEST_NAME,
    namespace: NAMESPACE,
    status_key: "stage",
    status_value: "started",
  }, {
    timeout: "1m",
    interval: "10s",
  });

  expect(err, "wait for started returns").toBeNull();

  let allPods = env.getN("pods", {
    "namespace": NAMESPACE,
    "app": "k6",
    "k6_cr": TEST_NAME,
  });

  let runnerPods = env.getN("pods", {
    "namespace": NAMESPACE,
    "app": "k6",
    "k6_cr": TEST_NAME,
    "runner": "true",
  });

  // there should be N runners pods + initializer + starter
  expect(runnerPods, "runner pod count").toBe(4);
  expect(allPods, "total pod count").toBe(runnerPods + 2);

  err = env.wait({
    kind: "TestRun",
    name: TEST_NAME,
    namespace: NAMESPACE,
    status_key: "stage",
    status_value: "finished",
  }, {
    timeout: "5m",
    interval: "10s",
  });

  // TODO: add check for status of the pods

  expect(err, "wait for finished returns").toBeNull();
}

export function teardown() {
  console.log("delete returns", env.delete());
}
