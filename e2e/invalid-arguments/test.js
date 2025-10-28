import { Environment } from 'k6/x/environment';
import { sleep, fail } from 'k6';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "./"

const env = new Environment({
  name: "invalid-arguments",
  implementation: "vcluster",
  initFolder: PARENT + "manifests", // initial folder with everything that will be loaded at init
})

export function setup() {
  console.log("init returns", env.init());
  // it is best to have a bit of delay between creating a CRD and 
  // a corresponding CR, so as to avoid the "no matches" error
  sleep(0.5);
}

// TestRun should enter error stage on invalid .spec.arguments
export default function () {
  let err = env.apply(PARENT + "testrun.yaml");
  console.log("apply testrun returns", err)

  err = env.wait({
    kind: "TestRun",
    name: "k6-sample", //tr.name(),
    namespace: "default",  //tr.namespace(),
    status_key: "stage",
    status_value: "error",
  }, {
    timeout: "5m",
    interval: "1m",
  });

  if (err != null) {
    fail("wait returns" + err);
  }
}

export function teardown() {
  console.log("delete returns", env.delete());
}