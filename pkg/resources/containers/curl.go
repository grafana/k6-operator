package containers

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	resource "k8s.io/apimachinery/pkg/api/resource"
)

// NewCurlContainer is used to get a template for a new k6 starting curl container.
func NewCurlContainer(hostnames []string, image string) corev1.Container {
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
		parts = append(parts, fmt.Sprintf("curl -X PATCH -H 'Content-Type: application/json' http://%s:6565/v1/status -d '%s'", hostname, req))
	}

	return corev1.Container{
		Name:  "k6-curl",
		Image: image,
		Env: []corev1.EnvVar{{
			Name:  "ISTIO_QUIT_API",
			Value: "http://127.0.0.1:15020",
		},
			{
				Name:  "ENVOY_ADMIN_API",
				Value: "http://localhost:15000",
			},
			{
				Name:  "WAIT_FOR_ENVOY_TIMEOUT",
				Value: "15",
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewMilliQuantity(50, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(2097152, resource.BinarySI),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: *resource.NewQuantity(209715200, resource.BinarySI),
			},
		},
		Command: []string{
			"scuttle",
			"sh",
			"-c",
			strings.Join(parts, ";"),
		},
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
