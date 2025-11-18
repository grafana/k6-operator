# k6-operator

![Version: 4.1.1](https://img.shields.io/badge/Version-4.1.1-informational?style=flat-square) ![AppVersion: 1.1.0](https://img.shields.io/badge/AppVersion-1.1.0-informational?style=flat-square)

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
| customAnnotations | object | `{}` | Custom Annotations to be applied on all resources |
| customLabels | object | `{}` | Custom Label to be applied on all resources |
| fullnameOverride | string | `""` |  |
| global.image | object | `{"pullSecrets":[],"registry":""}` | Global image configuration |
| global.image.pullSecrets | list | `[]` | Optional set of global image pull secrets |
| global.image.registry | string | `""` | Global image registry to use if it needs to be overridden for some specific use cases (e.g local registries, custom images, ...) |
| installCRDs | bool | `true` | Installs CRDs as part of the release |
| manager | object | `{"containerSecurityContext":{},"env":[],"envFrom":[],"image":{"pullPolicy":"IfNotPresent","registry":"ghcr.io","repository":"grafana/k6-operator","tag":"controller-v1.1.0"},"livenessProbe":{"httpGet":{"path":"/healthz","port":8081},"initialDelaySeconds":15,"periodSeconds":20},"logging":{"development":true},"podSecurityContext":{},"readinessProbe":{"httpGet":{"path":"/healthz","port":8081},"initialDelaySeconds":5,"periodSeconds":10},"replicas":1,"resources":{"limits":{"cpu":"100m","memory":"100Mi"},"requests":{"cpu":"100m","memory":"50Mi"}},"serviceAccount":{"create":true,"name":"k6-operator-controller"}}` | controller-manager configuration |
| manager.containerSecurityContext | object | `{}` | A security context defines privileges and access control settings for the container. |
| manager.env | list | `[]` | List of environment variables to set in the controller |
| manager.envFrom | list | `[]` | List of sources to populate environment variables in the controller |
| manager.image | object | `{"pullPolicy":"IfNotPresent","registry":"ghcr.io","repository":"grafana/k6-operator","tag":"controller-v1.1.0"}` | controller-manager image configuration |
| manager.image.pullPolicy | string | `"IfNotPresent"` | pull policy for the image possible values Always, Never, IfNotPresent (default: IfNotPresent) |
| manager.image.repository | string | `"grafana/k6-operator"` | controller-manager image repository |
| manager.image.tag | string | `"controller-v1.1.0"` | controller-manager image tag |
| manager.livenessProbe | object | `{"httpGet":{"path":"/healthz","port":8081},"initialDelaySeconds":15,"periodSeconds":20}` | Liveness probe in Probe format |
| manager.livenessProbe.httpGet | object | `{"path":"/healthz","port":8081}` | HTTP liveness probe |
| manager.logging | object | `{"development":true}` | controller-manager logging configuration |
| manager.logging.development | bool | `true` | Set to true to enable development mode logging (human-readable console format). Set to false for production mode logging (JSON format). |
| manager.podSecurityContext | object | `{}` | A security context defines privileges and access control settings for a pod. |
| manager.readinessProbe | object | `{"httpGet":{"path":"/healthz","port":8081},"initialDelaySeconds":5,"periodSeconds":10}` | Readiness probe in Probe format |
| manager.readinessProbe.httpGet | object | `{"path":"/healthz","port":8081}` | HTTP readiness probe |
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
| metrics.serviceMonitor.enabled | bool | `false` |  |
| metrics.serviceMonitor.honorLabels | bool | `false` |  |
| metrics.serviceMonitor.interval | string | `""` |  |
| metrics.serviceMonitor.jobLabel | string | `""` |  |
| metrics.serviceMonitor.labels | object | `{}` |  |
| metrics.serviceMonitor.metricRelabelings | list | `[]` |  |
| metrics.serviceMonitor.namespace | string | `""` |  |
| metrics.serviceMonitor.relabelings | list | `[]` |  |
| metrics.serviceMonitor.scrapeTimeout | string | `""` |  |
| metrics.serviceMonitor.selector | object | `{}` |  |
| nameOverride | string | `""` |  |
| namespace | object | `{"create":true}` | Namespace creation |
| namespace.create | bool | `true` | create the namespace (default: true) |
| nodeSelector | object | `{}` | Node Selector to be applied on all containers |
| podAnnotations | object | `{}` | Custom Annotations to be applied on all pods |
| podLabels | object | `{}` | Custom Label to be applied on all pods |
| rbac | object | `{"namespaced":false}` | RBAC configuration |
| rbac.namespaced | bool | `false` | If true, does not install cluster RBAC resources |
| service.annotations | object | `{}` | service custom annotations |
| service.enabled | bool | `true` | enables the k6-operator service (default: false) |
| service.labels | object | `{}` | service custom labels |
| service.portName | string | `"https"` | Name for controller-manager HTTP port |
| tolerations | list | `[]` | Tolerations to be applied on all containers |

