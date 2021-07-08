package jobs

import (
	"fmt"

	"github.com/k6io/operator/api/v1alpha1"
	"github.com/k6io/operator/pkg/resources/containers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewStarterJob builds a template used for creating a starter job
func NewStarterJob(k6 *v1alpha1.K6, ips []string) *batchv1.Job {

	starterAnnotations := make(map[string]string)
	if k6.Spec.Starter.Metadata.Annotations != nil {
		starterAnnotations = k6.Spec.Starter.Metadata.Annotations
	}

	starterLabels := newLabels(k6.Name)
	if k6.Spec.Starter.Metadata.Labels != nil {
		for k, v := range k6.Spec.Starter.Metadata.Labels { // Order not specified
			if _, ok := starterLabels[k]; !ok {
				starterLabels[k] = v
			}
		}
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-starter", k6.Name),
			Namespace: k6.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      starterLabels,
					Annotations: starterAnnotations,
				},
				Spec: corev1.PodSpec{
					Affinity:      k6.Spec.Starter.Affinity,
					NodeSelector:  k6.Spec.Starter.NodeSelector,
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						containers.NewCurlContainer(ips),
					},
				},
			},
		},
	}
}
