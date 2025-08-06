# Update of dependencies

## Golang dependencies

There are three main types of dependencies in the k6 Operator project.

1. `controller-runtime` and other Kubernetes dependencies are updated approximately once per quarter. This is in line with the Kubernetes release cycle.
    - This update is done with the manual review of all key changelogs by the maintainer, in order to ensure that any breaking changes in the dependencies are handled gracefully by the k6 Operator.
    - We might automate this update in the future (let a bot open a PR), but this change will always need careful monitoring.

2. The rest of Golang packages are updated more regularly, with the goal to automate updates down to a bi-weekly basis eventually.

3. Security patches are automated already and are merged the quickest.

## Kubebuilder update

Kubebuilder update can happen but it is pretty rare: so far, such an update happened only once, in June of 2025. It is not required for the k6 Operator to remain functional, as Kubebuilder is not its dependency. The change of Kubebuilder mainly means a change of default layout and scaffolding. OTOH, there are benefits to it, as such an update helps to stay up-to-date with approaches in the ecosystem.

It is unlikely that the Kubebuilder update will be needed often: it takes years for the Kubebuilder to release new versions. But whenever such update happens next time, it should be planned ahead as it is a large and complex update, requiring a lot of attention.

# Kubernetes compatibility

The Kubernetes version supported by the k6 Operator depends on the underlying libraries used to release it. As noted above, those libraries are being updated once per quarter approximately. If the latest version of Kubernetes is not yet supported by the k6 Operator, then likely it will be within the quarter.

The k6 Operator project tends to avoid adding dependencies on newer features of Kubernetes as much as possible and tries to use only older, stable interfaces. That means that in practice, k6 Operator can be deployed even on older versions of Kubernetes, for example 1.20. It must be noted that we do not recommend it and do not support it: if you do it, it's at your own risk.

# Releases

The k6 Operator does not have a strict release schedule at the moment: it is being released when there is a sufficient number of changes merged. In practice, it is released on an approximately bi-monthly schedule, without adherence to specific dates.

Release procedure is a bit manual and fully described in the [internal docs](https://github.com/grafana/k6-operator/blob/main/docs/releases.md). 

The `bundle.yaml` file is updated on a release only. It is not meant to be updated outside of the release.

Helm chart is released at the same time as the k6 Operator. By default, the new chart will have an increase of the minor version, unless there was a significant or breaking change. The chart can also be released separately from the app, with an increase of the patch version, if there's a need in bug fix.

TODO: v1
