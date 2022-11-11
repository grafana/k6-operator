package jobs

import (
	"fmt"
	"strconv"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/resources/containers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewStarterJob builds a template used for creating a starter job
func NewStarterJob(k6 *v1alpha1.K6, hostname []string) *batchv1.Job {

	starterAnnotations := make(map[string]string)
	if k6.Spec.Starter.Metadata.Annotations != nil {
		starterAnnotations = k6.Spec.Starter.Metadata.Annotations
	}

	starterImage := "ghcr.io/grafana/operator:latest-starter"
	if k6.Spec.Starter.Image != "" {
		starterImage = k6.Spec.Starter.Image
	}

	starterLabels := newLabels(k6.Name)
	if k6.Spec.Starter.Metadata.Labels != nil {
		for k, v := range k6.Spec.Starter.Metadata.Labels { // Order not specified
			if _, ok := starterLabels[k]; !ok {
				starterLabels[k] = v
			}
		}
	}
	serviceAccountName := "default"
	if k6.Spec.Starter.ServiceAccountName != "" {
		serviceAccountName = k6.Spec.Starter.ServiceAccountName
	}
	automountServiceAccountToken := true
	if k6.Spec.Starter.AutomountServiceAccountToken != "" {
		automountServiceAccountToken, _ = strconv.ParseBool(k6.Spec.Starter.AutomountServiceAccountToken)
	}

	command, istioEnabled := newIstioCommand(k6.Spec.Scuttle.Enabled, []string{"sh", "-c"})
	env := newIstioEnvVar(k6.Spec.Scuttle, istioEnabled)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-starter", k6.Name),
			Namespace:   k6.Namespace,
			Labels:      starterLabels,
			Annotations: starterAnnotations,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      starterLabels,
					Annotations: starterAnnotations,
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &automountServiceAccountToken,
					ServiceAccountName:           serviceAccountName,
					Affinity:                     k6.Spec.Starter.Affinity,
					NodeSelector:                 k6.Spec.Starter.NodeSelector,
					Tolerations:                  k6.Spec.Starter.Tolerations,
					RestartPolicy:                corev1.RestartPolicyNever,
					SecurityContext:              &k6.Spec.Starter.SecurityContext,
					ImagePullSecrets:             k6.Spec.Starter.ImagePullSecrets,
					Containers: []corev1.Container{
						containers.NewCurlContainer(hostname, starterImage, k6.Spec.Starter.ImagePullPolicy, command, env),
					},
				},
			},
		},
	}
}
