# k6-operator

![Version: 3.10.1](https://img.shields.io/badge/Version-3.10.1-informational?style=flat-square) ![AppVersion: 0.0.18](https://img.shields.io/badge/AppVersion-0.0.18-informational?style=flat-square)

A Helm chart to install the k6-operator

**Homepage:** <https://k6.io>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| yorugac | <olha@k6.io> |  |

## Source Code

* <https://github.com/grafana/k6-operator>

## Requirements

Kubernetes: `>=1.16.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity to be applied on all containers |
| authProxy.containerSecurityContext | object | `{}` | A security context defines privileges and access control settings for the container. |
| authProxy.enabled | bool | `true` | enables the protection of /metrics endpoint. (https://github.com/brancz/kube-rbac-proxy) |
| authProxy.image.pullPolicy | string | `"IfNotPresent"` | pull policy for the image can be Always, Never, IfNotPresent (default: IfNotPresent) |
| authProxy.image.registry | string | `"gcr.io"` |  |
| authProxy.image.repository | string | `"kubebuilder/kube-rbac-proxy"` | rbac-proxy image repository |
| authProxy.image.tag | string | `"v0.15.0"` | rbac-proxy image tag |
| authProxy.livenessProbe | object | `{}` | Liveness probe in Probe format |
| authProxy.readinessProbe | object | `{}` | Readiness probe in Probe format |
| authProxy.resources | object | `{}` | rbac-proxy resource limitation/request |
| customAnnotations | object | `{}` | Custom Annotations to be applied on all resources |
| customLabels | object | `{}` | Custom Label to be applied on all resources |
| global.image | object | `{"pullSecrets":[],"registry":""}` | Global image configuration |
| global.image.pullSecrets | list | `[]` | Optional set of global image pull secrets |
| global.image.registry | string | `""` | Global image registry to use if it needs to be overridden for some specific use cases (e.g local registries, custom images, ...) |
| installCRDs | bool | `true` | Installs CRDs as part of the release |
| manager | object | `{"containerSecurityContext":{},"env":[],"envFrom":[],"image":{"pullPolicy":"IfNotPresent","registry":"ghcr.io","repository":"grafana/k6-operator","tag":"controller-v0.0.18"},"livenessProbe":{},"podSecurityContext":{},"readinessProbe":{},"replicas":1,"resources":{"limits":{"cpu":"100m","memory":"100Mi"},"requests":{"cpu":"100m","memory":"50Mi"}},"serviceAccount":{"create":true,"name":"k6-operator-controller"}}` | controller-manager configuration |
| manager.containerSecurityContext | object | `{}` | A security context defines privileges and access control settings for the container. |
| manager.env | list | `[]` | List of environment variables to set in the controller |
| manager.envFrom | list | `[]` | List of sources to populate environment variables in the controller |
| manager.image | object | `{"pullPolicy":"IfNotPresent","registry":"ghcr.io","repository":"grafana/k6-operator","tag":"controller-v0.0.18"}` | controller-manager image configuration |
| manager.image.pullPolicy | string | `"IfNotPresent"` | pull policy for the image possible values Always, Never, IfNotPresent (default: IfNotPresent) |
| manager.image.repository | string | `"grafana/k6-operator"` | controller-manager image repository |
| manager.image.tag | string | `"controller-v0.0.18"` | controller-manager image tag |
| manager.livenessProbe | object | `{}` | Liveness probe in Probe format |
| manager.podSecurityContext | object | `{}` | A security context defines privileges and access control settings for a pod. |
| manager.readinessProbe | object | `{}` | Readiness probe in Probe format |
| manager.replicas | int | `1` | number of controller-manager replicas (default: 1) |
| manager.resources | object | `{"limits":{"cpu":"100m","memory":"100Mi"},"requests":{"cpu":"100m","memory":"50Mi"}}` | controller-manager Resources definition |
| manager.resources.limits | object | `{"cpu":"100m","memory":"100Mi"}` | controller-manager Resources limits |
| manager.resources.limits.cpu | string | `"100m"` | controller-manager CPU limit (Max) |
| manager.resources.limits.memory | string | `"100Mi"` | controller-manager Memory limit (Max) |
| manager.resources.requests | object | `{"cpu":"100m","memory":"50Mi"}` | controller-manager Resources requests |
| manager.resources.requests.cpu | string | `"100m"` | controller-manager CPU request (Min) |
| manager.resources.requests.memory | string | `"50Mi"` | controller-manager Memory request (Min) |
| manager.serviceAccount.create | bool | `true` | create the service account (default: true) |
| manager.serviceAccount.name | string | `"k6-operator-controller"` | kubernetes service account for the k6 manager |
| namespace | object | `{"create":true}` | Namespace creation |
| namespace.create | bool | `true` | create the namespace (default: true) |
| nodeSelector | object | `{}` | Node Selector to be applied on all containers |
| podAnnotations | object | `{}` | Custom Annotations to be applied on all pods |
| podLabels | object | `{}` | Custom Label to be applied on all pods |
| prometheus.enabled | bool | `false` | enables the prometheus metrics scraping (default: false) |
| tolerations | list | `[]` | Tolerations to be applied on all containers |

