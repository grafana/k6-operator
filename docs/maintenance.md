# Maintenance of k6 Operator

Just as with any project, k6 Operator requires regular maintenance updates to stay up-to-date with any changes and patches. There are several types of maintenance updates in this project, and they need to be addressed at different frequencies. Let's take a closer look at each of them.

## Update of dependencies

### Golang dependencies

There are three main types of Golang dependencies in the k6 Operator project.

1. `controller-runtime` and other Kubernetes dependencies are updated approximately once per quarter. This is in line with the Kubernetes release cycle.
    - In k6 Operator, this update is done with the manual review of all key changelogs by the maintainer, to ensure that any breaking changes in the dependencies are handled gracefully by the k6 Operator.
    - We might automate this update in the future (let a bot open a PR), but this change will always need careful monitoring.

2. The rest of the Golang packages are updated more regularly. We're likely to automate these updates with a bi-weekly schedule eventually.

3. Security patches are automated already and are merged the quickest.

### `controller-tools` update

[`controller-tools`](https://github.com/kubernetes-sigs/controller-tools) are essential for generation of manifests. They have their own versioning, separate from Kubernetes versioning. At the moment of writing, this project also does not have a stable release schedule.

That said, it makes sense to check on their updates approximately once per quarter or two and update as needed.

### Kubebuilder update

Update of Kubebuilder in the k6 Operator can happen as well, but it is most rare in practice. So far, such an update has happened only once, in June of 2025. It is not required for the k6 Operator to remain functional, as Kubebuilder is not its dependency. The change of Kubebuilder mainly means a change of the default layout and scaffolding. On the other hand, there are benefits to it, as such update helps to stay up-to-date with approaches in the Kubernetes ecosystem.

It is unlikely that the Kubebuilder update will be needed often: it takes years for the Kubebuilder to release new versions. But whenever such an update happens next time, it should be planned ahead of time, as it is a large and complex task, requiring a lot of attention and potentially resulting in breaking changes.

## Kubernetes compatibility

The Kubernetes version supported by the k6 Operator depends on the underlying libraries used to release it. As noted above, those libraries are being updated once per quarter, approximately. If the latest version of Kubernetes is not yet supported by the k6 Operator, then likely it will be within the quarter.

The k6 Operator project tends to avoid adding dependencies on newer features of Kubernetes as much as possible and tries to use only older, stable interfaces. That means that in practice, k6 Operator can be deployed even on older versions of Kubernetes, for example, 1.20. It must be noted that we do not recommend it and do not support it: if you do it, it's at your own risk.

If the above approach changes for some reason, we'll inform the community in GitHub issues and in release notes.

