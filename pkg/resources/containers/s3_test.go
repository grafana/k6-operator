package containers

import (
	"testing"

	deep "github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func Test_NewS3InitContainer(t *testing.T) {
	vm := corev1.VolumeMount{Name: "archive-volume", MountPath: "/test"}

	tests := []struct {
		name     string
		uri      string
		image    string
		expected v1alpha1.InitContainer
	}{
		{
			name:  "valid S3 URI",
			uri:   "https://bucket.s3.amazonaws.com/archive.tar",
			image: "curlimages/curl:latest",
			expected: v1alpha1.InitContainer{
				Name:         "archive-download",
				Image:        "curlimages/curl:latest",
				Command:      []string{"sh", "-c", "curl -X GET -L 'https://bucket.s3.amazonaws.com/archive.tar' > /test/archive.tar ; ls -l /test"},
				VolumeMounts: []corev1.VolumeMount{vm},
			},
		},
		{
			name:  "empty URI is not validated here",
			uri:   "",
			image: "image:latest",
			expected: v1alpha1.InitContainer{
				Name:         "archive-download",
				Image:        "image:latest",
				Command:      []string{"sh", "-c", "curl -X GET -L '' > /test/archive.tar ; ls -l /test"},
				VolumeMounts: []corev1.VolumeMount{vm},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewS3InitContainer(tt.uri, tt.image, vm)
			if diff := deep.Equal(got, tt.expected); diff != nil {
				t.Errorf("NewS3InitContainer() diff: %s", diff)
			}
		})
	}
}
