package testrun

import (
	"fmt"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/resources/containers"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestName(testRunId string) string {
	return fmt.Sprintf("plz-test-%s", testRunId)
}

// ingestURL is a temp hack
func NewPLZTestRun(plz *v1alpha1.PrivateLoadZone, trData *cloud.TestRunData, ingestUrl string) *v1alpha1.K6 {
	volume := corev1.Volume{
		Name: "archive-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      "archive-volume",
		MountPath: "/test",
	}

	initContainer := containers.NewS3InitContainer(
		trData.ArchiveURL,
		"ghcr.io/grafana/k6-operator:latest-starter",
		volumeMount,
	)

	return &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestName(trData.TestRunID()),
			Namespace: plz.Namespace,
		},
		Spec: v1alpha1.K6Spec{
			Runner: v1alpha1.Pod{
				Image:              trData.RunnerImage,
				ServiceAccountName: plz.Spec.ServiceAccountName,
				NodeSelector:       plz.Spec.NodeSelector,
				Resources: corev1.ResourceRequirements{
					Limits: plz.Spec.Resources,
					// Requests will default to the Limits values.
				},
				Volumes: []corev1.Volume{
					volume,
				},
				VolumeMounts: []corev1.VolumeMount{
					volumeMount,
				},
				InitContainers: []v1alpha1.InitContainer{
					initContainer,
				},
				Env: []corev1.EnvVar{{
					Name:  "K6_CLOUD_HOST",
					Value: ingestUrl,
				}},
			},
			Starter: v1alpha1.Pod{
				ServiceAccountName: plz.Spec.ServiceAccountName,
				NodeSelector:       plz.Spec.NodeSelector,
			},
			Script: v1alpha1.K6Script{
				LocalFile: "/test/archive.tar",
			},
			Parallelism: int32(trData.InstanceCount),
			Separate:    true,
			Arguments:   "--out cloud --no-thresholds",
			Cleanup:     v1alpha1.Cleanup("post"),

			TestRunID: trData.TestRunID(),
			Token:     plz.Spec.Token,
		},
	}
}
