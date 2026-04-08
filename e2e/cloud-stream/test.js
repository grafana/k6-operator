import { Environment } from 'k6/x/environment';
import { sleep, fail } from 'k6';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "./"

const env = new Environment({
  name: "cloud-stream",
  implementation: "vcluster",
  initFolder: PARENT + "manifests",
})

export function setup() {
  console.log("init returns", env.init());
  sleep(0.5);
}

export default function () {
  let err = env.apply(PARENT + "testrun.yaml");
  console.log("apply testrun returns", err);

  err = env.wait({
    kind: "TestRun",
    name: "k6-sample",
    namespace: "k6-tests",
    status_key: "stage",
    status_value: "started",
  }, {
    timeout: "1m",
    interval: "10s",
  });

  if (err != null) {
    fail("wait for started returns " + err);
  }

  let allPods = env.getN("pods", {
    "namespace": "k6-tests",
    "app": "k6",
    "k6_cr": "k6-sample",
  });

  let runnerPods = env.getN("pods", {
    "namespace": "k6-tests",
    "app": "k6",
    "k6_cr": "k6-sample",
    "runner": "true",
  });

  // there should be N runner pods + initializer + starter
  if (runnerPods != 1 || allPods != runnerPods + 2) {
    fail("wrong number of pods:" + runnerPods + "/" + allPods);
  }

  err = env.wait({
    kind: "TestRun",
    name: "k6-sample",
    namespace: "k6-tests",
    status_key: "stage",
    status_value: "finished",
  }, {
    timeout: "5m",
    interval: "10s",
  });

  if (err != null) {
    fail("wait for finished returns " + err);
  }
}

export function teardown() {
  console.log("delete returns", env.delete());
}