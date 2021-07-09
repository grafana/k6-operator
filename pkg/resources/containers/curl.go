package containers

import (
	"encoding/json"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

// NewCurlContainer is used to get a template for a new k6 starting curl container.
func NewCurlContainer(ips []string) v1.Container {
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

	return v1.Container{
		Name:  "k6-curl",
		Image: "radial/busyboxplus:curl",
		Command: []string{
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
