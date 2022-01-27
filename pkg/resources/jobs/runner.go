package jobs

import (
	"errors"
	"fmt"
	"strconv"

	"strings"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/segmentation"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Internal script type created from Spec.script possible options
type Script struct {
	Name string
	File string
	Type string
}

// NewRunnerJob creates a new k6 job from a CRD
func NewRunnerJob(k6 *v1alpha1.K6, index int) (*batchv1.Job, error) {
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

	script, err := newScript(k6.Spec)

	if err != nil {
		return nil, err
	}

	if k6.Spec.Arguments != "" {
		args := strings.Split(k6.Spec.Arguments, " ")
		command = append(command, args...)
	}

	command = append(
		command,
		fmt.Sprintf(script.File),
		"--address=0.0.0.0:6565")

	paused := true
	if k6.Spec.Paused != "" {
		paused, _ = strconv.ParseBool(k6.Spec.Paused)
	}

	if paused {
		command = append(command, "--paused")
	}

	command = appendFileCheckerCommand(script, command)

	var zero int64 = 0

	image := "ghcr.io/grafana/operator:latest-runner"
	if k6.Spec.Runner.Image != "" {
		image = k6.Spec.Runner.Image
	}

	runnerAnnotations := make(map[string]string)
	if k6.Spec.Runner.Metadata.Annotations != nil {
		runnerAnnotations = k6.Spec.Runner.Metadata.Annotations
	}

	runnerLabels := newLabels(k6.Name)
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
	env = append(env, k6.Spec.Runner.Env...)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   k6.Namespace,
			Labels:      runnerLabels,
			Annotations: runnerAnnotations,
		},
		Spec: batchv1.JobSpec{
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
					Containers: []corev1.Container{{
						Image:        image,
						Name:         "k6",
						Command:      command,
						Env:          env,
						Resources:    k6.Spec.Runner.Resources,
						VolumeMounts: newVolumeMountSpec(script),
						Ports:        ports,
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       newVolumeSpec(script),
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

func newVolumeMountSpec(s *Script) []corev1.VolumeMount {
	if s.Type == "LocalFile" {
		return []corev1.VolumeMount{}
	}
	return []corev1.VolumeMount{{
		Name:      "k6-test-volume",
		MountPath: "/test",
	}}
}

func newVolumeSpec(s *Script) []corev1.Volume {
	switch s.Type {
	case "VolumeClaim":
		return []corev1.Volume{{
			Name: "k6-test-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: s.Name,
				},
			},
		}}
	case "ConfigMap":
		return []corev1.Volume{{
			Name: "k6-test-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: s.Name,
					},
				},
			},
		}}
	default:
		return []corev1.Volume{}
	}
}

func newScript(spec v1alpha1.K6Spec) (*Script, error) {
	s := &Script{}
	s.File = "test.js"

	if spec.Script.VolumeClaim.Name != "" {
		s.Name = spec.Script.VolumeClaim.Name
		if spec.Script.VolumeClaim.File != "" {
			s.File = spec.Script.VolumeClaim.File
		}

		s.File = fmt.Sprintf("/test/%s", s.File)
		s.Type = "VolumeClaim"
		return s, nil
	}

	if spec.Script.ConfigMap.Name != "" {
		s.Name = spec.Script.ConfigMap.Name

		if spec.Script.ConfigMap.File != "" {
			s.File = spec.Script.ConfigMap.File
		}
		s.File = fmt.Sprintf("/test/%s", s.File)
		s.Type = "ConfigMap"
		return s, nil
	}

	if spec.Script.LocalFile != "" {
		s.Name = "LocalFile"
		s.File = spec.Script.LocalFile
		s.Type = "LocalFile"
		return s, nil
	}

	return nil, errors.New("ConfigMap, VolumeClaim or LocalFile not provided in script definition")
}

func appendFileCheckerCommand(s *Script, cmd []string) []string {
	if s.Type == "LocalFile" {
		joincmd := strings.Join(cmd, " ")
		checkCommand := []string{"sh", "-c", fmt.Sprintf("if [ ! -f %v ]; then echo \"LocalFile not found exiting...\"; exit 1; fi;\n%v", s.File, joincmd)}
		return checkCommand
	}
	return cmd
}
