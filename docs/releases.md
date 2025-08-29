# Versioning and releases

There are three types of versions in k6 Operator:
1. Version of the k6 Operator app, released as a [GitHub tag](https://github.com/grafana/k6-operator/releases) in the `k6-operator` repo.
1. Version of each of the CRD types supported by k6 Operator. They follow [Kubernetes versioning](https://kubernetes.io/docs/reference/using-api/#api-versioning).
1. Version of the Helm chart that installs k6 Operator and is published to the [`grafana/helm-charts` repo](https://github.com/grafana/helm-charts).

We must clearly distinguish between these three types. However, when we speak about "k6 Operator version", we usually mean the first type: the version of the k6 Operator app. If something else is intended, it will be noted explicitly.

## Versioning

Until 1.0.0, k6 Operator was released with versioning fit for experimental software, for example, as 0.0.23. This does not allow for evaluating the complexity of the release without reading the release notes. Additionally, there were some RC releases in the past, too, further muddying the picture.

With the 1.0.0 release, k6 Operator is committing to [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html). Releases are now supposed to happen regularly, with either a minor version increase or a major version increase.

### Major version

Release with an increase of the major version can happen in the following cases:
1. Version upgrade of the existing CRD type.
1. Major backwards incompatible change of the existing CRD type, no matter its version.
    - Note, minor breaking changes in the CRD type will not warrant a major release if the following is true:
      - The CRD type is at `v1alpha1` version.
      - The change is breaking but minor in scope; for example, a change of type [from string to boolean](https://github.com/grafana/k6-operator/issues/455).
1. Major change in functionality of the application.
1. Other major changes as decided by internal Grafana Labs priorities.

### Minor version

Release with an increase of the minor version happens on a schedule. Refer to the [next section](#release-schedule) for details.

Such a release can contain additions of functionality or backward-compatible changes to existing functionality.

### Patch version

Release with an increase of the patch version happens when there are critical bugs or urgent security updates.

### Helm versioning

Versioning of the Helm chart is separate from the k6 Operator version, for historical reasons, but it also follows Semantic Versioning.

The Helm chart is always released right after the k6 Operator release. It means that an increase in the k6 Operator version increases the corresponding Helm chart version.

By default, the new chart will have an increase of the minor version, unless there was a significant or breaking change. The chart can also be released separately from the k6 Operator app. For example, if there's a need for a bug fix specific to the chart, the Helm chart is released with an increase in the patch version.

A major increase in the k6 Operator version will result in a major increase in the corresponding Helm chart.

## Release schedule

Until 1.0.0, the k6 Operator did not have a strict release schedule. Instead, it was released when there was a sufficient number of changes merged. In practice, it was released on an approximately bi-monthly schedule, without adherence to specific dates.

With the 1.0.0 release, k6 Operator is formally committing to releasing a major or minor version every **8 weeks**. Additionally, we no longer care about the number of changes in a release. For example, if there's only a dependency update merged into the `main` branch by the end of 8 weeks, it will still be released as the next minor version.

Major version release is not scheduled ahead. However, once there is a decision to have a major release of k6 Operator at some point in the foreseeable future, it will be communicated to the community in GitHub and in the release notes of the minor version release.

Patch version release happens on an as-needed basis.

### Stability guarantees

#### CRD types

As mentioned in the [versioning description](#major-version), _major_ breaking changes of any CRD type will result in major release of the k6 Operator.

If the CRD is at `v1alpha1` version and if there are _minor_ breaking changes in this CRD, they will be released as a minor version of the k6 Operator. In this case, there will be no support for two versions: users are expected to adjust their CRD files as part of the upgrade of the k6 Operator. We classify "minor" here as changing the format of 1-2 lines in the spec.

Update of a CRD's version will be added in a backwards compatible way first, supporting both versions, with a deprecation warning about the older version of CRD. Once the next major release of k6 Operator is done, the older version of the CRD may be removed.

#### `bundle.yaml`

The [`bundle.yaml`](https://github.com/grafana/k6-operator/blob/main/bundle.yaml) file is updated on a release only. It is not meant to be updated outside of the release. So it is guaranteed to be stable and correspond to the latest released version of k6 Operator.

#### Command-line options

Command-line option to k6 Operator changes very rarely in practice. If there's a breaking change in command-line options, it should be released with an increase in the major version. If there's an addition or extension of command-line options, it can be released as a minor version.

#### `main` branch

The `main` branch of the k6 Operator should be seen as a development branch. It is not guaranteed to be stable outside of tagged releases.

#### Golang interfaces

We do not commit to any stability guarantees of Golang interfaces at this point in time. However, in practice, the [Golang package for CRD types](https://github.com/grafana/k6-operator/tree/main/api/v1alpha1) is closely tied to changes in CRD and follows the same guidelines as the corresponding CRD type.

### Release process

Release procedure is fully described in the [internal docs](https://github.com/grafana/k6-operator/blob/main/docs/release-workflow.md).

