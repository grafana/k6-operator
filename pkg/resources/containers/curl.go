package containers

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	resource "k8s.io/apimachinery/pkg/api/resource"
)

// NewCurlContainer is used to get a template for a new k6 starting curl container.
func NewCurlContainer(hostnames []string, image string, command []string, env []corev1.EnvVar) corev1.Container {
	req, _ := json.Marshal(
		statusAPIRequest{
			Data: statusAPIRequestData{
				Attributes: statusAPIRequestDataAttributes{
					Paused: false,
				},
				ID:   "default",
				Type: "status",
			},
		})

	var parts []string
	for _, hostname := range hostnames {
		parts = append(parts, fmt.Sprintf("curl --retry 3 -X PATCH -H 'Content-Type: application/json' http://%s:6565/v1/status -d '%s'", hostname, req))
	}

	return corev1.Container{
		Name:  "k6-curl",
		Image: image,
		Env:   env,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewMilliQuantity(50, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(2097152, resource.BinarySI),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(209715200, resource.BinarySI),
			},
		},
		Command: append(
			command,
			strings.Join(parts, ";"),
		),
	}
}

type statusAPIRequest struct {
	Data statusAPIRequestData `json:"data"`
}

type statusAPIRequestData struct {
	Attributes statusAPIRequestDataAttributes `json:"attributes"`
	ID         string                         `json:"id"`
	Type       string                         `json:"type"`
}

type statusAPIRequestDataAttributes struct {
	Paused bool `json:"paused"`
}
