import { Environment } from 'k6/x/environment';
import { sleep, fail } from 'k6';

export const options = {
    setupTimeout: '60s',
};

const PARENT = "./"

const env = new Environment({
    name: "init-container-volume",
    implementation: "vcluster",
    initFolder: PARENT + "manifests", // initial folder with everything that wil be loaded at init
})

export function setup() {
    console.log("init returns", env.init());
    // it is best to have a bit of delay between creating a CRD and 
    // a corresponding CR, so as to avoid the "no matches" error
    sleep(0.5);
}

// This test checks at once init container and volumes for runners,
// as well as localFile option.
export default function () {
    let err = env.apply(PARENT + "testrun.yaml");
    console.log("apply testrun returns", err)

    // ideally, we should check pod spec here, but this test
    // will never finish successfully without init volume working 
    // as expected as there won't be any script to execute

    err = env.wait({
        kind: "TestRun",
        name: "k6-init-container", //tr.name(),
        namespace: "default",      //tr.namespace(),
        status_key: "stage",
        status_value: "finished",
    }, {
        timeout: "5m",
        interval: "1m",
    });

    if (err != null) {
        fail("wait returns" + err);
    }

    let allPods = env.getN("pods", {
        "namespace": "default", //tr.namespace()
        "app": "k6",
        "k6_cr": "k6-init-container", //tr.name()
    });

    // there should be N runners pods + initializer + starter
    if (allPods != 2 + 2) {
        fail("wrong number of pods:" + allPods + " instead of " + 4);
    }
}

export function teardown() {
    console.log("delete returns", env.delete());
}