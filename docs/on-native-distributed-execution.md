# Distributed k6 and k6-operator

This document is my long overdue write-up on the design doc for [distributed execution in k6](https://github.com/grafana/k6/blob/master/docs/design/020-distributed-execution-and-test-suites.md) and what it means for k6-operator. That design doc is heavily referenced here and will be addressed as "NDE design doc" for simplicity. (NDE stands for native distributed execution.) It is strongly advised to be acquainted with it for understanding of the current doc.

## Does k6 OSS support distributed mode now?

Some believe that k6 OSS does not support the distributed mode now. It is not fully correct: k6 OSS _can_ be executed in some version of a distributed mode even now, using the [execution segments](https://grafana.com/docs/k6/latest/using-k6/k6-options/reference/#execution-segment). E.g. one can write a very small custom script that bootstraps k6 distributed execution and it will work for many cases. But as is mentioned in the NDE design doc, there are some [missing parts](https://github.com/grafana/k6/blob/master/docs/design/020-distributed-execution-and-test-suites.md#missing-parts-in-the-current-state) of a full distributed execution (a list of 9 points) that are not supported by k6 OSS **out-of-the-box**. Arguably, even a good chunk of these missing parts (with possible exception of metric aggregation) can be circumvented with additional, rather small tooling if one really wants. AFAIK, there have been some folk who have gone exactly this route, writing custom tooling to run distributed k6. 

The point here is that the missing parts list is not rocket science™. OTOH, the need to write that additional tooling can be seen as cumbersome. And the cost of maintenance of this tooling long-term may be too much, depending on the use case one tries to support.

I'm using the term "tooling for distributed mode" to describe these additional software layers that are needed to fill the "missing parts" gaps when running k6 in distributed mode now. They can be implemented in different ways, from scripts to services. We often use the term "native distributed execution" to basically mean the implementation native to k6: that is, when most or all the tooling for distributed mode is implemented as part of k6 OSS.

## Why should we care?

I omit the matter of the "most upvoted [k6 issue](https://github.com/grafana/k6/issues/140)", in order to focus on the matter of design of our own (Grafana) products.

Given the previous section, it should come as no surprise that we in actuality have and support our own implementations of tooling for distributed mode. These implementations exist in parallel, they are partial in terms of missing parts list and tailored for the chosen specific use cases. Specifically:
1. implementation at Grafana Cloud k6 (closed sourced). It's not supporting the whole [list of 9 points](https://github.com/grafana/k6/blob/master/docs/design/020-distributed-execution-and-test-suites.md#missing-parts-in-the-current-state) yet (or not until recently? TODO: needs clarification). But most importantly, it is targetting the specific case of current implementation of GCk6 which is tied to [AWS instances](https://grafana.com/docs/grafana-cloud/testing/k6/reference/cloud-ips/#cloud-ip-list).
2. implementation in k6-operator (OSS). It has a similar partial support of [9 points list](https://github.com/grafana/k6/blob/master/docs/design/020-distributed-execution-and-test-suites.md#missing-parts-in-the-current-state) as GCk6 but *only for Cloud tests*, in order to achieve feature-parity for the customers. Due to Kubernetes abstractions, from some view point it can be seen as more flexible than GCk6 one (in that Kubernetes implementation can be ported relatively quickly to popular clouds), but less flexible from another (those abstractions tie one to how Kubernetes orchestrates executions but it's not the only way to orchestrate them). But the lack of the same feature parity for OSS test runs creates an unnecessary complexity in this current implementation and it'd be good to minimize the difference between Cloud and OSS.
3. [PR](https://github.com/grafana/k6/pull/2816) of native implementation in k6 (OSS): it is a draft but it was reported as working and, most importantly, as open for possibilities of full implementation and further enhancements.

IOW, there is a PR for NDE in k6 OSS and we currently support two different implementations for distributed k6, simply because of the need to implement tooling for distributed mode on top of k6. Those implementations are non-ideal and, worse, an attempt to merge them into one is unlikely to result in a good solution. More on that in the next section.

### Can our existing implementations be merged into one?

I've been pondering this from time to time and mostly came to conclusion that it doesn't make much sense at this point of time.

Firstly, k6-operator is a Kubernetes operator first. So by its very nature, it is an app that exists within certain limitations of Kubernetes platform, like etcd providing the only persistance out-of-the-box. Even though there are ways to circumvent these limitations in _general case_, doing so here might make k6-operator as good as unusable for existing users. It is unlikely a good idea to over-complicate the k6-operator as an OSS product. Somewhat similarly to the reasoning brought up in the [alternative solutions section in the NDE design doc](https://github.com/grafana/k6/blob/master/docs/design/020-distributed-execution-and-test-suites.md#alternative-solutions), k6-operator would be better off to remain as a light-weight Kubernetes interface to distributed k6 rather than anything heavy and tied to additional technologies. Though I'd argue it is not sufficiently light-weight even now; more on that in the next section.

While k6-operator already duplicates some logic of the GCk6 implementation, an attempt to duplicate it in full, specifically for the [setup / teardown issue](https://github.com/grafana/k6-operator/issues/223), brought mixed results. What is relatively "natural" to do to a k6 process with, say, a Bash script (e.g. terminate it with a [SIGTERM](https://github.com/grafana/k6/issues/3583)) can only be seen as a hack when done by a Kubernetes operator. Kubernetes itself is a set of abstractions over processes, and to implement a GCk6-like teardown we would have to break that abstraction by going 'down' the abstraction ladder to exert direct control. This can hardly be called a good design. Moreover, it would be hard to predict the long-term consequences of such solution in any, existing and future, Kubernetes cluster it might end up operating.

I think Kubernetes operators should not be breaking the existing status quo of abstraction layers and trying to play "services" or low-level "controllers" of other applications. Instead, it is much better for the application itself (k6 OSS) to provide a simple UX for the required functionality. Then, the operator simply uses that UX without much ado and remains simple and light-weight itself. This way the operator becomes the "glue" between the app and Kubernetes world, and the app remains self-contained and independent from Kubernetes.

The NDE design doc also alludes to top-down implementation of synchronization being part of the problem:
> Secondly, a big architectural problem of the current distributed execution solutions is that they have a push-based and top-down control mechanism. [..] This limited control mechanism is what's actually causing the need to re-implement some of the current k6 logic externally (e.g. top-down instance synchronization, setup() and teardown() handling, etc.). 

This echoes my observations above: by moving the necessary logic (tooling for distributed mode) to application (k6), it is possible to keep all the higher-level abstractions clean, easy to understand and maintain. AFAIS, any other solution will likely create a mess, even if it works for some use case.

Just for completeness sake, I should point out that generally speaking, there are two ways to merge existing implementations:
1. by moving (re-implementing, adapting, etc.) parts of GCk6 implementation into k6-operator: the problems with it are described above.
2. by using k6-operator as GCk6 implementation, which sadly does not make much sense as:
    - ATM, k6-operator solves a narrower set of use cases than GCk6 (more on that in the section after next).
    - The implementation of GCk6 is rather strongly tailored for its use case on AWS and Kubernetes implementation does not map to it directly.

I also think that the k6-operator implementation would still benefit from simplification, regardless of where it's used. I outlined some reasons for this above, and will add more in the next section.

### Why would NDE help the k6-operator?

Some notable points about implementation of tooling for distributed mode in k6-operator and how NDE in k6 would impact it:
- k6-operator acts as a test coordinator now.
    - With NDE, all or most of the coordination logic will be removed from it.
- The connection between runners (agents) and k6-operator is not [reversed](https://github.com/grafana/k6/blob/master/docs/design/020-distributed-execution-and-test-suites.md#reverse-the-pattern) and trying to reverse it would go against the natural flow of a Kubernetes operator. Alternatively it would require deployment of additional entities.
    - With NDE, k6-operator can rely on the coordinator to provide info about state of the test and stop connecting to the runners directly at all.
- Validation of a script requires bootstrapping a separate k6 process.
    - With NDE, validation will hopefully become part of the coordinator's reponsibilities.
- All HTTP calls to GCk6 now happen as part of k6-operator.
    - With NDE, at least some of the HTTP calls to GCk6 will become responsibility of the coordinator and will be removed from k6-operator.
    - Subsequently, the difference in logic between OSS and Cloud test runs will also be minimized. This is not only about providing an OSS solution (like support for setup / teardown) but about simplifying the underlying code: there will be no need to have multiple scenarios at the level of k6-operator. The most complex issue k6-operator ever had, idempotency, was precisely because of this point.
- Passing of the script is now native to Kubernetes, e.g. mounting a ConfigMap to the runners.
    - With NDE, the same pattern can still be supported but limited to a coordinator pod, if need be.
- k6-operator is currently creating a Job for each k6 execution.
    - The benefit of this is questionable, other than potential for [node failures](https://kubernetes.io/docs/concepts/workloads/controllers/job/#completion-mode) but OTOH, there was never any explicit user request to move from Jobs to simple Pods.
    - NDE doesn't care which implementation will be used. But a move to NDE can be used to implement this change as well or at least propose it to the community. Specifically: coordinator is created as a Job with a Service attached (so that k6-operator can communicate with it) and agents as Pods and they connect to coordinator themselves.

NDE might also unlock an easier implementation of some features in k6-operator, like support of externally-controlled executor.

All in all, it is safe to claim that NDE in k6 will make k6-operator truly light-weight, simpler and therefore easier to maintain. It will help us achieve the "feature completeness" for the TestRun CRD: the state at which it requires minimum maintenance, mostly coming down to updating dependencies on the schedule.

### What about multiple regions support?

Currently k6-operator solves a narrower set of use cases in comparison to GCk6 because GCk6 supports execution in [multiple regions at once](https://grafana.com/docs/grafana-cloud/testing/k6/author-run/use-load-zones/#syntax-to-set-load-zones) while k6-operator is tied to one Kubernetes cluster (also, [multi-cluster execution issue](https://github.com/grafana/k6-operator/issues/240)).

Support of multi-cluster in Kubernetes is a long-debated problem. There is hope of some form of multi-cluster support being added to Kubernetes natively. When it is added, we definitely will be able to use it in k6-operator, but such changes move at a glacial speed and may not cover the case of k6 execution in multiple regions.

There are also several non-native (external) solutions for multi-cluster in Kubernetes nowadays. But dragging one or another multi-cluster solution into k6-operator is unlikely to be a good idea. Ideally, k6-operator should not depend on any one multi-cluster setup but instead be pluggable into whatever cluster and networking setup is in place. And here is where native distributed mode in k6 might come in handy too:

The architecture of NDE in the draft [PR](https://github.com/grafana/k6/pull/2816) allows to have multiple levels of coordinators. The multiple regions execution can be seen as requiring two levels of coordinators: a coordinator per region on the first level (mostly acting as relays of data for minimization of network traffic in-between regions) and one central coordinator on the second level. (also described [here](https://github.com/grafana/k6/pull/3217#issuecomment-1924280133)) This approach allows to have theoretically any configuration of k6 processes, including the operator bootstrapping these jobs in any region / cluster, while also keeping the solution independent from the details of cluster setup.

## Conclusion

The points above boil down to the following: while it is theoretically possible to implement the tooling for distributed mode for almost any use case, it doesn't mean it should be done if there is a way to avoid it. In order to avoid it, one needs native distributed execution in k6. Specifically for us in Grafana k6, native distributed execution would greatly help to merge our existing implementations of distributed k6 while also achieving a clean bottom-up design.

IMO, k6 OSS should be a self-contained tool which can be integrated into other workflows natively, be it AWS, Kubernetes, CI pipeline or Bash scripting on a local machine. If we need a distributed version of k6 (and we do, obviously, looking at GCk6 and k6-operator), then it only makes sense for k6 to support the distributed mode natively and provide similar UX to that of k6 standalone mode. Then, any other implementation, be it a light-weight k6-operator or a GCk6 platform, can be built more naturally on top of k6 OSS without jumping through the hoops as we do now.

Also, important to point out: if tooling for distributed mode is part of k6 (native distributed execution), we will likely reduce the maintenance surface in half and condense it to mostly k6 itself.
