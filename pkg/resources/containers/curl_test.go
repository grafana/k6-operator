package containers

import (
	"testing"

	deep "github.com/go-test/deep"
	corev1 "k8s.io/api/core/v1"
)

func Test_NewStartContainer(t *testing.T) {
	startPayload := `{"data":{"attributes":{"paused":false,"stopped":false},"id":"default","type":"status"}}`

	tests := []struct {
		name     string
		hosts    []string
		image    string
		expected corev1.Container
	}{
		{
			name:  "1 hostname",
			hosts: []string{"test-1"},
			image: "curlimages/curl:latest",
			expected: corev1.Container{
				Name:            "k6-curl",
				Image:           "curlimages/curl:latest",
				ImagePullPolicy: corev1.PullNever,
				Env:             []corev1.EnvVar{{Name: "K6_VAR", Value: "val"}},
				Resources:       corev1.ResourceRequirements{},
				Command: []string{"sh", "-c",
					"curl --retry 3 -X PATCH -H 'Content-Type: application/json' http://test-1:6565/v1/status -d '" + startPayload + "' -s -w '\n{\"http_code\":%{http_code},\"time_total\":%{time_total},\"time_starttransfer\":%{time_starttransfer},\"url\":\"%{url_effective}\",\"remote_ip\":\"%{remote_ip}\",\"errormsg\":\"%{errormsg}\"}'",
				},
				SecurityContext: &corev1.SecurityContext{},
			},
		},
		{
			name:  "N hostnames",
			hosts: []string{"test-1", "test-2"},
			image: "curlimages/curl:latest",
			expected: corev1.Container{
				Name:            "k6-curl",
				Image:           "curlimages/curl:latest",
				ImagePullPolicy: corev1.PullNever,
				Env:             []corev1.EnvVar{{Name: "K6_VAR", Value: "val"}},
				Resources:       corev1.ResourceRequirements{},
				Command: []string{"sh", "-c",
					"curl --retry 3 -X PATCH -H 'Content-Type: application/json' http://test-1:6565/v1/status -d '" + startPayload + "' -s -w '\n{\"http_code\":%{http_code},\"time_total\":%{time_total},\"time_starttransfer\":%{time_starttransfer},\"url\":\"%{url_effective}\",\"remote_ip\":\"%{remote_ip}\",\"errormsg\":\"%{errormsg}\"}';" +
						"curl --retry 3 -X PATCH -H 'Content-Type: application/json' http://test-2:6565/v1/status -d '" + startPayload + "' -s -w '\n{\"http_code\":%{http_code},\"time_total\":%{time_total},\"time_starttransfer\":%{time_starttransfer},\"url\":\"%{url_effective}\",\"remote_ip\":\"%{remote_ip}\",\"errormsg\":\"%{errormsg}\"}'",
				},
				SecurityContext: &corev1.SecurityContext{},
			},
		},
	}

	env := []corev1.EnvVar{{Name: "K6_VAR", Value: "val"}}
	cmd := []string{"sh", "-c"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStartContainer(tt.hosts, tt.image, corev1.PullNever, cmd, env, corev1.SecurityContext{}, corev1.ResourceRequirements{})
			if diff := deep.Equal(got, tt.expected); diff != nil {
				t.Errorf("NewStartContainer() diff: %s", diff)
			}
		})
	}
}

func Test_NewStopContainer(t *testing.T) {
	stopPayload := `{"data":{"attributes":{"paused":false,"stopped":true},"id":"default","type":"status"}}`

	tests := []struct {
		name     string
		hosts    []string
		image    string
		expected corev1.Container
	}{
		{
			name:  "1 hostname",
			hosts: []string{"test-1"},
			image: "curlimages/curl:latest",
			expected: corev1.Container{
				Name:            "k6-curl",
				Image:           "curlimages/curl:latest",
				ImagePullPolicy: corev1.PullNever,
				Resources:       corev1.ResourceRequirements{},
				Command: []string{"sh", "-c",
					"curl --retry 3 -X PATCH -H 'Content-Type: application/json' http://test-1:6565/v1/status -d '" + stopPayload + "' -s -w '\n{\"http_code\":%{http_code},\"time_total\":%{time_total},\"time_starttransfer\":%{time_starttransfer},\"url\":\"%{url_effective}\",\"remote_ip\":\"%{remote_ip}\",\"errormsg\":\"%{errormsg}\"}'",
				},
				SecurityContext: &corev1.SecurityContext{},
			},
		},
		{
			name:  "N hostnames",
			hosts: []string{"test-1", "test-2"},
			image: "curlimages/curl:latest",
			expected: corev1.Container{
				Name:            "k6-curl",
				Image:           "curlimages/curl:latest",
				ImagePullPolicy: corev1.PullNever,
				Resources:       corev1.ResourceRequirements{},
				Command: []string{"sh", "-c",
					"curl --retry 3 -X PATCH -H 'Content-Type: application/json' http://test-1:6565/v1/status -d '" + stopPayload + "' -s -w '\n{\"http_code\":%{http_code},\"time_total\":%{time_total},\"time_starttransfer\":%{time_starttransfer},\"url\":\"%{url_effective}\",\"remote_ip\":\"%{remote_ip}\",\"errormsg\":\"%{errormsg}\"}';" +
						"curl --retry 3 -X PATCH -H 'Content-Type: application/json' http://test-2:6565/v1/status -d '" + stopPayload + "' -s -w '\n{\"http_code\":%{http_code},\"time_total\":%{time_total},\"time_starttransfer\":%{time_starttransfer},\"url\":\"%{url_effective}\",\"remote_ip\":\"%{remote_ip}\",\"errormsg\":\"%{errormsg}\"}'",
				},
				SecurityContext: &corev1.SecurityContext{},
			},
		},
	}

	cmd := []string{"sh", "-c"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStopContainer(tt.hosts, tt.image, corev1.PullNever, cmd, nil, corev1.SecurityContext{}, corev1.ResourceRequirements{})
			if diff := deep.Equal(got, tt.expected); diff != nil {
				t.Errorf("NewStopContainer() diff: %s", diff)
			}
		})
	}
}
