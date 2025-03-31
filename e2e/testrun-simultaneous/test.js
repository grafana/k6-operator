import { Environment } from 'k6/x/environment';
import { sleep, fail } from 'k6';

import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "./"

const env = new Environment({
  name: "testrun-simultaneous",
  implementation: "vcluster",
  initFolder: PARENT + "manifests", // initial folder with everything that wil be loaded at init
})

export function setup() {
  console.log("init returns", env.init());
  // it is best to have a bit of delay between creating a CRD and 
  // a corresponding CR, so as to avoid the "no matches" error
  sleep(0.5);
}

// A test to check simultaneous execution of 2 tests is successful.
export default function () {
  let err = env.apply(PARENT + "testrun1.yaml");
  console.log("apply testrun1 returns", err)

  err = env.apply(PARENT + "testrun2.yaml");
  console.log("apply testrun2 returns", err)

  const r = randomIntBetween(1, 2);

  // randomize order of the check as it shouldn't matter
  if (r > 1) {
    wait_for_second(env);
    wait_for_first(env);
  } else {
    wait_for_first(env);
    wait_for_second(env);
  }
}

export function teardown() {
  console.log("delete returns", env.delete());
}

function wait_for_first(env) {
  let err = env.wait({
    kind: "TestRun",
    name: "t-2-runners", //tr1.name(),
    namespace: "default",  //tr1.namespace(),
    status_key: "stage",
    status_value: "finished",
  }, {
    timeout: "5m",
    interval: "1m",
  });

  if (err != null) {
    fail("wait for t-2-runners returns" + err);
  }

  let allPods = env.getN("pods", {
    "namespace": "default", //tr.namespace()
    "app": "k6",
    "k6_cr": "t-2-runners", //tr.name()
  });

  // there should be N runners pods + initializer + starter
  if (allPods != 2 + 2) {
    fail("wrong number of pods:" + allPods + " instead of " + 4);
  }
}

function wait_for_second(env) {
  let err = env.wait({
    kind: "TestRun",
    name: "t-3-runners", //tr2.name(),
    namespace: "default",  //tr2.namespace(),
    status_key: "stage",
    status_value: "finished",
  }, {
    timeout: "5m",
    interval: "1m",
  });

  if (err != null) {
    fail("wait for t-3-runners returns" + err);
  }

  let allPods = env.getN("pods", {
    "namespace": "default", //tr.namespace()
    "app": "k6",
    "k6_cr": "t-3-runners", //tr.name()
  });

  // there should be N runners pods + initializer + starter
  if (allPods != 3 + 2) {
    fail("wrong number of pods:" + allPods + " instead of " + 5);
  }
}