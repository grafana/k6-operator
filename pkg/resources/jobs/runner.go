package jobs

import (
	"fmt"
	"strconv"

	"strings"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/segmentation"
	"github.com/grafana/k6-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewRunnerJob creates a new k6 job from a CRD
func NewRunnerJob(k6 *v1alpha1.K6, index int, testRunId, token string) (*batchv1.Job, error) {
	name := fmt.Sprintf("%s-%d", k6.Name, index)
	postCommand := []string{"k6", "run"}

	command, istioEnabled := newIstioCommand(k6.Spec.Scuttle.Enabled, postCommand)

	quiet := true
	if k6.Spec.Quiet != "" {
		quiet, _ = strconv.ParseBool(k6.Spec.Quiet)
	}

	if quiet {
		command = append(command, "--quiet")
	}

	if k6.Spec.Parallelism > 1 {
		var args []string
		var err error

		if args, err = segmentation.NewCommandFragments(index, int(k6.Spec.Parallelism)); err != nil {
			return nil, err

		}
		command = append(command, args...)
	}

	script, err := types.ParseScript(&k6.Spec)
	if err != nil {
		return nil, err
	}

	if k6.Spec.Arguments != "" {
		args := strings.Split(k6.Spec.Arguments, " ")
		command = append(command, args...)
	}

	command = append(
		command,
		fmt.Sprintf(script.FullName()),
		"--address=0.0.0.0:6565")

	paused := true
	if k6.Spec.Paused != "" {
		paused, _ = strconv.ParseBool(k6.Spec.Paused)
	}

	if paused {
		command = append(command, "--paused")
	}

	// Add an instance tag: in case metrics are stored, they need to be distinguished by instance
	command = append(command, "--tag", fmt.Sprintf("instance_id=%d", index))

	// Add an job tag: in case metrics are stored, they need to be distinguished by job
	command = append(command, "--tag", fmt.Sprintf("job_name=%s", name))

	command = script.UpdateCommand(command)

	var (
		zero   int64 = 0
		zero32 int32 = 0
	)

	image := "ghcr.io/grafana/operator:latest-runner"
	if k6.Spec.Runner.Image != "" {
		image = k6.Spec.Runner.Image
	}

	runnerAnnotations := make(map[string]string)
	if k6.Spec.Runner.Metadata.Annotations != nil {
		runnerAnnotations = k6.Spec.Runner.Metadata.Annotations
	}

	runnerLabels := newLabels(k6.Name)
	runnerLabels["runner"] = "true"
	if k6.Spec.Runner.Metadata.Labels != nil {
		for k, v := range k6.Spec.Runner.Metadata.Labels { // Order not specified
			if _, ok := runnerLabels[k]; !ok {
				runnerLabels[k] = v
			}
		}
	}

	serviceAccountName := "default"
	if k6.Spec.Runner.ServiceAccountName != "" {
		serviceAccountName = k6.Spec.Runner.ServiceAccountName
	}

	automountServiceAccountToken := true
	if k6.Spec.Runner.AutomountServiceAccountToken != "" {
		automountServiceAccountToken, _ = strconv.ParseBool(k6.Spec.Runner.AutomountServiceAccountToken)
	}

	ports := []corev1.ContainerPort{{ContainerPort: 6565}}
	ports = append(ports, k6.Spec.Ports...)

	env := newIstioEnvVar(k6.Spec.Scuttle, istioEnabled)

	// this is a cloud output run
	if len(testRunId) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "K6_CLOUD_PUSH_REF_ID",
			Value: testRunId,
		}, corev1.EnvVar{
			Name:  "K6_CLOUD_TOKEN",
			Value: token,
		})
	}

	env = append(env, k6.Spec.Runner.Env...)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   k6.Namespace,
			Labels:      runnerLabels,
			Annotations: runnerAnnotations,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &zero32,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      runnerLabels,
					Annotations: runnerAnnotations,
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &automountServiceAccountToken,
					ServiceAccountName:           serviceAccountName,
					Hostname:                     name,
					RestartPolicy:                corev1.RestartPolicyNever,
					Affinity:                     k6.Spec.Runner.Affinity,
					NodeSelector:                 k6.Spec.Runner.NodeSelector,
					Tolerations:                  k6.Spec.Runner.Tolerations,
					SecurityContext:              &k6.Spec.Runner.SecurityContext,
					ImagePullSecrets:             k6.Spec.Runner.ImagePullSecrets,
					Containers: []corev1.Container{{
						Image:           image,
						ImagePullPolicy: k6.Spec.Runner.ImagePullPolicy,
						Name:            "k6",
						Command:         command,
						Env:             env,
						Resources:       k6.Spec.Runner.Resources,
						VolumeMounts:    script.VolumeMount(),
						Ports:           ports,
						EnvFrom:         k6.Spec.Runner.EnvFrom,
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}

	if k6.Spec.Separate {
		job.Spec.Template.Spec.Affinity = newAntiAffinity()
	}
	return job, nil
}

func NewRunnerService(k6 *v1alpha1.K6, index int) (*corev1.Service, error) {
	serviceName := fmt.Sprintf("%s-%s-%d", k6.Name, "service", index)
	runnerName := fmt.Sprintf("%s-%d", k6.Name, index)

	runnerAnnotations := make(map[string]string)
	if k6.Spec.Runner.Metadata.Annotations != nil {
		runnerAnnotations = k6.Spec.Runner.Metadata.Annotations
	}

	runnerLabels := newLabels(k6.Name)
	runnerLabels["runner"] = "true"
	if k6.Spec.Runner.Metadata.Labels != nil {
		for k, v := range k6.Spec.Runner.Metadata.Labels { // Order not specified
			if _, ok := runnerLabels[k]; !ok {
				runnerLabels[k] = v
			}
		}
	}

	port := []corev1.ServicePort{{
		Name:     "http-api",
		Port:     6565,
		Protocol: "TCP",
	}}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        serviceName,
			Namespace:   k6.Namespace,
			Labels:      runnerLabels,
			Annotations: runnerAnnotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: port,
			Selector: map[string]string{
				"job-name": runnerName,
			},
		},
	}

	return service, nil
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
