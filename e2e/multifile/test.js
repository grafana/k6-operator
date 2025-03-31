import { Environment } from 'k6/x/environment';
import { sleep, fail } from 'k6';

export const options = {
    setupTimeout: '60s',
};

const PARENT = "./"

const env = new Environment({
    name: "multifile",
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

    err = env.wait({
        kind: "TestRun",
        name: "multifile-test", //tr.name(),
        namespace: "default",  //tr.namespace(),
        status_key: "stage",
        status_value: "finished",
    }, {
        timeout: "2m",
        interval: "1s",
    });
    if (err != null) {
        fail("wait returns" + err);
    }

    let allPods = env.getN("pods", {
        "namespace": "default", // tr.namespace()
        "app": "k6",
        "k6_cr": "multifile-test", //tr.name()
    });

    // there should be N runners pods + initializer + starter
    if (allPods != 3 + 2) {
        fail("wrong number of pods:" + allPods + " instead of " + 5);
    }
}

export function teardown() {
    console.log("delete returns", env.delete());
}