package jobs

import (
	"fmt"

	"strings"

	"github.com/k6io/operator/api/v1alpha1"
	"github.com/k6io/operator/pkg/segmentation"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewRunnerJob creates a new k6 job from a CRD
func NewRunnerJob(k6 *v1alpha1.K6, index int) (*batchv1.Job, error) {
	name := fmt.Sprintf("%s-%d", k6.Name, index)
	command := []string{"k6", "run", "--quiet"}

	if k6.Spec.Parallelism > 1 {
		var args []string
		var err error

		if args, err = segmentation.NewCommandFragments(index, int(k6.Spec.Parallelism)); err != nil {
			return nil, err

		}
		command = append(command, args...)
	}

	if k6.Spec.Arguments != "" {
		args := strings.Split(k6.Spec.Arguments, " ")
		command = append(command, args...)
	}
	command = append(
		command,
		"/test/test.js",
		"--address=0.0.0.0:6565",
		"--paused")

	var zero int64 = 0

	image := "loadimpact/k6:latest"
	if k6.Spec.Image != "" {
		image = k6.Spec.Image
	}

	runnerAnnotations := make(map[string]string)
	if k6.Spec.Runner.Annotations != nil {
		runnerAnnotations = k6.Spec.Runner.Annotations
	}

	runnerLabels := newLabels(k6.Name)
	if k6.Spec.Runner.Labels != nil {
		for k, v := range k6.Spec.Runner.Labels { // Order not specified
			if _, ok := runnerLabels[k]; !ok {
				runnerLabels[k] = v
			}
		}
	}

	ports := []corev1.ContainerPort{{ContainerPort: 6565}}
	ports = append(ports, k6.Spec.Ports...)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k6.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      runnerLabels,
					Annotations: runnerAnnotations,
				},
				Spec: corev1.PodSpec{
					Hostname:      name,
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Image:   image,
						Name:    "k6",
						Command: command,
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "k6-test-volume",
							MountPath: "/test",
						}},
						Ports: ports,
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       newVolumeSpec(k6.Spec.Script),
				},
			},
		},
	}

	if k6.Spec.Separate {
		job.Spec.Template.Spec.Affinity = newAntiAffinity()
	}
	return job, nil
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
