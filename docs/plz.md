# Private Load Zone under the hood

Private Load Zone (PLZ) feature requires k6-operator to communicate regularly with Grafana Cloud k6 (GCk6) API. This document aims to describe the data flow of that communication.

## PLZ lifecycle

PLZ resource is explicitly created and explicitly destroyed by the user, e.g. with standard `kubectl` tooling. Once PLZ resource is created, k6-operator registers it with GCk6 and starts a polling loop. GCk6 is polled each 10 seconds for new test runs to be executed on the PLZ:

```mermaid
sequenceDiagram
    participant GCk6 API
    participant Kubernetes
    participant k6-operator
    actor Alice (SRE)
    Alice (SRE)->>Kubernetes: kubectl apply -f plz.yaml
    k6-operator->>GCk6 API: Register PLZ
    loop Polling for PLZ test runs: 10sec

        k6-operator->>GCk6 API: Get test runs for PLZ
        activate GCk6 API
        GCk6 API-->>k6-operator: List of test run IDs
        deactivate GCk6 API

    end

    Alice (SRE)->>Kubernetes: kubectl delete -f plz.yaml
    k6-operator->>GCk6 API: Deregister PLZ
```

In other words, there are three HTTP REST calls to GCk6 in the above workflow: to register, to deregister and to poll test runs. More on GCk6 REST API can be found [here](https://grafana.com/docs/grafana-cloud/k6/reference/cloud-rest-api/#read-test-runs).

## Lifecycle of PLZ test run

When a user starts any GCk6 test run, k6 first creates an [archive](https://grafana.com/docs/k6/latest/misc/archive/#k6-cloud-execution) with it and sends it to GCk6. First GCk6 executes internal validation of the archive and, in case of PLZ test run, stores the archive to AWS S3 with the [presigned URL](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html) and expiration time set to 300 seconds. GCk6 then notifies k6-operator about this test run during the next polling check, as described [above](#plz-lifecycle).

Once k6-operator learns of a new PLZ test run, it requests additional info about this test run from GCk6. Test run data returned by GCk6 contains presigned URL to S3 bucket and some additional information, like public Docker image containing k6 (`grafana/k6`) and amount of runners needed for this test run. Using this information, k6-operator then creates a `TestRun` custom resource (CR). 

```mermaid
sequenceDiagram
    actor Bob (QA)
    participant k6
    participant GCk6 API
    participant AWS S3
    participant k6-operator
    participant Kubernetes

    Bob (QA)->>k6: k6 cloud plz-test-X.js
    k6->>GCk6 API: k6 archive for X
    activate GCk6 API
    GCk6 API->>AWS S3: Store k6 archive for X
    deactivate GCk6 API
    Note over k6-operator: Oh, I have a PLZ test run X!
        
    k6-operator->>GCk6 API: Get data for test run X
    activate GCk6 API
    GCk6 API-->>k6-operator: Data for test run X
    deactivate GCk6 API
    k6-operator->>Kubernetes: Create TestRun CR
    rect rgb(191, 223, 255)
        note right of k6-operator: Running PLZ test run
        %% create participant runners
        runners->>AWS S3: Download k6 archive for X
        k6-operator->>runners: Start the test
        loop Test Run execution
            runners->>GCk6 API: cloud output
            k6-operator->>GCk6 API: Get test run state
            activate GCk6 API
            GCk6 API-->>k6-operator: Running OK!
            deactivate GCk6 API
        end
    end
```

As the PLZ `TestRun` has presigned URL configured as a path to k6 script, each runner Pod will download k6 archive from this URL in init container. The PLZ `TestRun` is also configured as a [cloud output test run](https://grafana.com/docs/k6/latest/results-output/real-time/cloud/) so runners are streaming metrics to GCk6 for aggregation, storage and visualization. 

Otherwise, PLZ `TestRun` is processed by k6-operator as any other `TestRun`, but with two additional HTTP REST calls to GCk6:
- a call that checks if test run is being processed without error by GCk6 and whether there is a user abort
- _optional_ a call that sends events about errors to GCk6 in case the test cannot be executed (e.g. something is off with infrastructure)
