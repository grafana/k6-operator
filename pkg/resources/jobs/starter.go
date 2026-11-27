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
	starter := starterSpec(k6)

	var (
		starterImage                 = "ghcr.io/grafana/k6-operator:latest-starter"
		starterAnnotations           = make(map[string]string)
		starterLabels                = newLabels(k6.NamespacedName().Name)
		serviceAccountName           = "default"
		automountServiceAccountToken = true
		schedulerName                = corev1.DefaultSchedulerName
		// Default resource requests and limits to use as a fallback
		resourceRequirements = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewMilliQuantity(50, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(2097152, resource.BinarySI),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(209715200, resource.BinarySI),
			},
		}
	)

	if starter.Metadata.Annotations != nil {
		starterAnnotations = starter.Metadata.Annotations
	}

	if starter.Image != "" {
		starterImage = starter.Image
	}

	if starter.Metadata.Labels != nil {
		for k, v := range starter.Metadata.Labels { // Order not specified
			if _, ok := starterLabels[k]; !ok {
				starterLabels[k] = v
			}
		}
	}
	if starter.ServiceAccountName != "" {
		serviceAccountName = starter.ServiceAccountName
	}
	if starter.AutomountServiceAccountToken != "" {
		automountServiceAccountToken, _ = strconv.ParseBool(starter.AutomountServiceAccountToken)
	}

	command, istioEnabled := newIstioCommand(k6.GetSpec().Scuttle.Enabled, []string{"sh", "-c"})
	env := newIstioEnvVar(k6.GetSpec().Scuttle, istioEnabled)

	// User specified resource requirements
	if len(starter.Resources.Requests) > 0 || len(starter.Resources.Limits) > 0 {
		resourceRequirements = starter.Resources
	}

	if starter.SchedulerName != "" {
		schedulerName = starter.SchedulerName
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
					Affinity:                     starter.Affinity,
					NodeSelector:                 starter.NodeSelector,
					Tolerations:                  starter.Tolerations,
					TopologySpreadConstraints:    starter.TopologySpreadConstraints,
					RestartPolicy:                corev1.RestartPolicyNever,
					SchedulerName:                schedulerName,
					SecurityContext:              &starter.SecurityContext,
					ImagePullSecrets:             starter.ImagePullSecrets,
					Containers: []corev1.Container{
						containers.NewStartContainer(
							hostname,
							starterImage,
							starter.ImagePullPolicy,
							command,
							env,
							starter.ContainerSecurityContext,
							resourceRequirements,
						),
					},
					PriorityClassName: starter.PriorityClassName,
				},
			},
		},
	}
}

// starterSpec returns the configuration of the starter Pod, which the stop Pod
// uses too, or the defaults if it is omitted.
func starterSpec(k6 *v1alpha1.TestRun) v1alpha1.Pod {
	if k6.GetSpec().Starter == nil {
		return v1alpha1.Pod{}
	}
	return *k6.GetSpec().Starter
}
