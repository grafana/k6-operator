import { Environment } from 'k6/x/environment';
import { sleep } from 'k6';
import { expect } from '../assertions.js';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "./"

const env = new Environment({
  name: "testrun-cloud-output",
  implementation: "vcluster",
  initFolder: PARENT + "manifests", // initial folder with everything that wil be loaded at init
})

export function setup() {
  console.log("init returns", env.init());
  // it is best to have a bit of delay between creating a CRD and 
  // a corresponding CR, so as to avoid the "no matches" error
  sleep(0.5);
}

export default function () {
  let err = env.apply(PARENT + "testrun.yaml");
  console.log("apply testrun returns", err);

  err = env.wait({
    kind: "TestRun",
    name: "k6-sample", //tr.name(),
    namespace: "k6-tests",  //tr.namespace(),
    status_key: "stage",
    status_value: "started",
  }, {
    timeout: "1m",
    interval: "10s",
  });

  expect(err, "wait for started returns").toBeNull();

  let allPods = env.getN("pods", {
    "namespace": "k6-tests", //tr.namespace()
    "app": "k6",
    "k6_cr": "k6-sample", //tr.name()
  });

  let runnerPods = env.getN("pods", {
    "namespace": "k6-tests", //tr.namespace()
    "app": "k6",
    "k6_cr": "k6-sample", //tr.name()
    "runner": "true",
  });

  // there should be N runners pods + initializer + starter
  expect(runnerPods, "runner pod count").toBe(4);
  expect(allPods, "total pod count").toBe(runnerPods + 2);

  err = env.wait({
    kind: "TestRun",
    name: "k6-sample", //tr.name(),
    namespace: "k6-tests",  //tr.namespace(),
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
