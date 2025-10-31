package jobs

import (
	"fmt"
	"strconv"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/resources/containers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewStarterJob builds a template used for creating a starter job
func NewStarterJob(k6 *v1alpha1.TestRun, hostname []string) *batchv1.Job {

	starterAnnotations := make(map[string]string)
	if k6.GetSpec().Starter.Metadata.Annotations != nil {
		starterAnnotations = k6.GetSpec().Starter.Metadata.Annotations
	}

	starterImage := "ghcr.io/grafana/k6-operator:latest-starter"
	if k6.GetSpec().Starter.Image != "" {
		starterImage = k6.GetSpec().Starter.Image
	}

	starterLabels := newLabels(k6.NamespacedName().Name)
	if k6.GetSpec().Starter.Metadata.Labels != nil {
		for k, v := range k6.GetSpec().Starter.Metadata.Labels { // Order not specified
			if _, ok := starterLabels[k]; !ok {
				starterLabels[k] = v
			}
		}
	}
	serviceAccountName := "default"
	if k6.GetSpec().Starter.ServiceAccountName != "" {
		serviceAccountName = k6.GetSpec().Starter.ServiceAccountName
	}
	automountServiceAccountToken := true
	if k6.GetSpec().Starter.AutomountServiceAccountToken != "" {
		automountServiceAccountToken, _ = strconv.ParseBool(k6.GetSpec().Starter.AutomountServiceAccountToken)
	}

	command, istioEnabled := newIstioCommand(k6.GetSpec().Scuttle.Enabled, []string{"sh", "-c"})
	env := newIstioEnvVar(k6.GetSpec().Scuttle, istioEnabled)

	// Default resource requests and limits to use as a fallback
	resourceRequirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(50, resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(2097152, resource.BinarySI),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(209715200, resource.BinarySI),
		},
	}

	// User specified resource requirements
	if len(k6.GetSpec().Starter.Resources.Requests) > 0 || len(k6.GetSpec().Starter.Resources.Limits) > 0 {
		resourceRequirements = k6.GetSpec().Starter.Resources
	}

	var zero32 int32

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-starter", k6.NamespacedName().Name),
			Namespace:   k6.NamespacedName().Namespace,
			Labels:      starterLabels,
			Annotations: starterAnnotations,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &zero32,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      starterLabels,
					Annotations: starterAnnotations,
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &automountServiceAccountToken,
					ServiceAccountName:           serviceAccountName,
					Affinity:                     k6.GetSpec().Starter.Affinity,
					NodeSelector:                 k6.GetSpec().Starter.NodeSelector,
					Tolerations:                  k6.GetSpec().Starter.Tolerations,
					TopologySpreadConstraints:    k6.GetSpec().Starter.TopologySpreadConstraints,
					RestartPolicy:                corev1.RestartPolicyNever,
					SecurityContext:              &k6.GetSpec().Starter.SecurityContext,
					ImagePullSecrets:             k6.GetSpec().Starter.ImagePullSecrets,
					Containers: []corev1.Container{
						containers.NewStartContainer(
							hostname,
							starterImage,
							k6.GetSpec().Starter.ImagePullPolicy,
							command,
							env,
							k6.GetSpec().Starter.ContainerSecurityContext,
							resourceRequirements,
						),
					},
					PriorityClassName: k6.GetSpec().Starter.PriorityClassName,
				},
			},
		},
	}
}
