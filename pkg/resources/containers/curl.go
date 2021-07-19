package containers

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// NewCurlContainer is used to get a template for a new k6 starting curl container.
func NewCurlContainer(ips []string) corev1.Container {
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
	for _, ip := range ips {
		parts = append(parts, fmt.Sprintf("curl -X PATCH -H 'Content-Type: application/json' http://%s:6565/v1/status -d '%s'", ip, req))
	}

	return corev1.Container{
		Name:  "k6-curl",
		Image: "radial/busyboxplus:curl",
		Command: []string{
			"sh",
			"-c",
			strings.Join(parts, ";"),
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
