import { Environment } from 'k6/x/environment';
import { sleep, fail } from 'k6';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "./"

const env = new Environment({
  name: "testrun-watch-namespaces",
  implementation: "vcluster",
  initFolder: PARENT + "manifests", // initial folder with everything that wil be loaded at init
})

export function setup() {
  console.log("init returns", env.init());
  // it is best to have a bit of delay between creating a CRD and 
  // a corresponding CR, so as to avoid the "no matches" error
  sleep(0.5);
}

// A test to check behaviour of WATCH_NAMESPACE
export default function () {
  let err = env.apply(PARENT + "testrun.yaml");
  console.log("apply testrun returns", err)

  err = env.apply(PARENT + "testrun-other.yaml");
  console.log("apply testrun-other returns", err)

  err = env.apply(PARENT + "testrun-invisible.yaml");
  console.log("apply testrun-invisible returns", err)

  err = env.wait({
    kind: "TestRun",
    name: "k6-sample", //tr1.name(),
    namespace: "some-ns",  //tr1.namespace(),
    status_key: "stage",
    status_value: "finished",
  }, {
    timeout: "5m",
    interval: "1m",
  });

  if (err != null) {
    fail("wait for k6-sample in some-ns returns" + err);
  }

  err = env.wait({
    kind: "TestRun",
    name: "k6-sample", //tr1.name(),
    namespace: "some-other-ns",  //tr1.namespace(),
    status_key: "stage",
    status_value: "finished",
  }, {
    timeout: "5m",
    interval: "1m",
  });

  if (err != null) {
    fail("wait for k6-sample in some-other-ns returns" + err);
  }

  err = env.wait({
    kind: "TestRun",
    name: "k6-sample", //tr2.name(),
    namespace: "invisible",  //tr2.namespace(),
    status_key: "stage",
    status_value: "", // stage here should never be populated by k6-operator
  }, {
    timeout: "5m",
    interval: "1m",
  });

  // Uncomment this, once this issue is done:
  // https://github.com/grafana/xk6-environment/issues/17
  // if (err !== null) {
  //   fail("wait for k6-sample in default returns" + err);
  // }
}

export function teardown() {
  console.log("delete returns", env.delete());
}
