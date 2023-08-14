package containers

import (
	"fmt"

	"github.com/grafana/k6-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// NewS3InitContainer is used to download a script archive from S3.
func NewS3InitContainer(uri, image string, volumeMount corev1.VolumeMount) v1alpha1.InitContainer {
	return v1alpha1.InitContainer{
		Name:         "archive-download",
		Image:        image,
		Command:      []string{"sh", "-c", fmt.Sprintf("curl -X GET -L '%s' > /test/archive.tar ; ls -l /test", uri)},
		VolumeMounts: []corev1.VolumeMount{volumeMount},
	}
}
