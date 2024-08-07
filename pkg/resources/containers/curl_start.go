package containers

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/grafana/k6-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"

	resource "k8s.io/apimachinery/pkg/api/resource"
)

// NewStartContainer is used to get a template for a new k6 starting curl container.
func NewStartContainer(hostnames []string, image string, imagePullPolicy corev1.PullPolicy, command []string, env []corev1.EnvVar, securityContext corev1.SecurityContext) corev1.Container {
	req, _ := json.Marshal(
		types.StatusAPIRequest{
			Data: types.StatusAPIRequestData{
				Attributes: types.StatusAPIRequestDataAttributes{
					Paused: false,
				},
				ID:   "default",
				Type: "status",
			},
		})

	var parts []string
	for _, hostname := range hostnames {
		parts = append(parts, fmt.Sprintf("curl --retry 3 -X PATCH -H 'Content-Type: application/json' http://%s/v1/status -d '%s'", net.JoinHostPort(hostname, "6565"), req))
	}

	return corev1.Container{
		Name:            "k6-curl",
		Image:           image,
		ImagePullPolicy: imagePullPolicy,
		Env:             env,
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
		SecurityContext: &securityContext,
	}
}
