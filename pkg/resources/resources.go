package resources

import (
	"fmt"
	"github.com/k6io/operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewJob creates a new k6 job from a CRD
func NewJob(k *v1alpha1.K6, index int) *batchv1.Job {
	name := fmt.Sprintf("%s-%d", k.Name, index)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: newLabels(k.Name),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Image:   "loadimpact/k6:latest",
						Name:    "k6",
						Command: []string{"k6", "run", "/test/test.js"},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "k6-test-volume",
							MountPath: "/test",
						}},
						Env: newEnvVars(k.Spec.Parallelism, index),
					}},
					Volumes: newVolumeSpec(k.Spec.Script),
				},
			},
		},
	}

	if k.Spec.Separate {
		job.Spec.Template.Spec.Affinity = newAntiAffinity()
	}

	return job
}

func newEnvVars(parallelism int32, index int) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "K6_INSTANCES_INDEX",
			Value: fmt.Sprintf("%d", index),
		},
		{
			Name:  "K6_INSTANCES_TOTAL",
			Value: fmt.Sprintf("%d", parallelism),
		},
	}
}

func newVolumeSpec(script string) []corev1.Volume {
	return []corev1.Volume{{
		Name: "k6-test-volume",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: script,
				},
			},
		},
	}}
}

func newAntiAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: "In",
								Values: []string{
									"k6",
								},
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}
}

func newLabels(name string) map[string]string {
	return map[string]string{
		"app":   "k6",
		"k6_cr": name,
	}
}
