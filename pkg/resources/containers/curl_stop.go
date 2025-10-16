package containers

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/grafana/k6-operator/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

// NewStopContainer is used to get a template for a new k6 stop curl container.
func NewStopContainer(hostnames []string, image string, imagePullPolicy corev1.PullPolicy, command []string, env []corev1.EnvVar, securityContext corev1.SecurityContext, resources corev1.ResourceRequirements) corev1.Container {
	req, _ := json.Marshal(
		types.StatusAPIRequest{
			Data: types.StatusAPIRequestData{
				Attributes: types.StatusAPIRequestDataAttributes{
					Stopped: true,
				},
				ID:   "default",
				Type: "status",
			},
		})

	var parts []string
	for _, hostname := range hostnames {
		parts = append(parts, fmt.Sprintf("curl --retry 3 -X PATCH -H 'Content-Type: application/json' http://%s/v1/status -d '%s' -s -w '\n{\"http_code\":%%{http_code},\"time_total\":%%{time_total},\"time_starttransfer\":%%{time_starttransfer},\"url\":\"%%{url_effective}\",\"remote_ip\":\"%%{remote_ip}\",\"errormsg\":\"%%{errormsg}\"}'", net.JoinHostPort(hostname, "6565"), req))
	}

	return corev1.Container{
		Name:            "k6-curl",
		Image:           image,
		ImagePullPolicy: imagePullPolicy,
		Env:             env,
		Resources:       resources,
		Command: append(
			command,
			strings.Join(parts, ";"),
		),
		SecurityContext: &securityContext,
	}
}
