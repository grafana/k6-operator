import { Environment } from 'k6/x/environment';
import { sleep, fail } from 'k6';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "./"

const env = new Environment({
  name: "testrun-cleanup",
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
  console.log("apply testrun returns", err)

  // let k6-operator read & bootstrap the TestRun
  sleep(10);

  let allPods = env.getN("pods", {
    "namespace": "fancy-testing",
    "app": "k6",
    "k6_cr": "k6-sample", //tr.name()
  });

  // there should be at least initializer pod by now
  if (allPods < 1) {
    fail("wrong number of pods: " + allPods);
  }

  err = env.wait({
    kind: "TestRun",
    name: "k6-sample", //tr.name(),
    namespace: "fancy-testing",  //tr.namespace(),
    status_key: "stage",
    status_value: "finished",
  }, {
    timeout: "2m",
    interval: "1s",
  });

  // Either wait() will "catch" TestRun at finished stage or
  // TestRun will be deleted some time between wait checks. Both
  // of those are valid in this case.
  if (err != null && err != "context deadline exceeded") {
    fail("unexpected error from wait(): " + err);
  }

  // there should be no pods at this point

  allPods = env.getN("pods", {
    "namespace": "fancy-testing", // tr.namespace()
    "app": "k6",
    "k6_cr": "k6-sample", //tr.name()
  });

  if (allPods != 0) {
    fail("pods were not deleted, there are " + allPods + " pods");
  }
}

export function teardown() {
  console.log("delete returns", env.delete());
}