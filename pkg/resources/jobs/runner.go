package jobs

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/segmentation"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewRunnerJob creates a new k6 job from a CRD
// secretName is the name of the Secret with Cloud token, which must be in the same namespace.
func NewRunnerJob(tr *v1alpha1.TestRun, index int, sti *cloud.SecretTokenInfo) (*batchv1.Job, error) {
	name := fmt.Sprintf("%s-%d", tr.NamespacedName().Name, index)
	postCommand := []string{"k6", "run"}

	command, istioEnabled := newIstioCommand(tr.GetSpec().Scuttle.Enabled, postCommand)

	quiet := true
	if tr.GetSpec().Quiet != "" {
		quiet, _ = strconv.ParseBool(tr.GetSpec().Quiet)
	}

	if quiet {
		command = append(command, "--quiet")
	}

	if tr.GetSpec().Parallelism > 1 {
		var args []string
		var err error

		if args, err = segmentation.NewCommandFragments(index, int(tr.GetSpec().Parallelism)); err != nil {
			return nil, err

		}
		command = append(command, args...)
	}

	script, err := tr.GetSpec().ParseScript()
	if err != nil {
		return nil, err
	}

	if tr.GetSpec().Arguments != "" {
		args := strings.Split(tr.GetSpec().Arguments, " ")
		command = append(command, args...)
	}

	command = append(
		command,
		script.FullName(),
		"--address=0.0.0.0:6565")

	paused := true
	if tr.GetSpec().Paused != "" {
		paused, _ = strconv.ParseBool(tr.GetSpec().Paused)
	}

	if paused {
		command = append(command, "--paused")
	}

	// Add an instance tag: in case metrics are stored, they need to be distinguished by instance
	command = append(command, "--tag", fmt.Sprintf("instance_id=%d", index))

	// Add a testrun name tag: in case metrics are stored, they need to be distinguished by test run name
	command = append(command, "--tag", fmt.Sprintf("testrun_name=%s", tr.NamespacedName().Name))

	if v1alpha1.IsTrue(tr, v1alpha1.CloudPLZTestRun) {
		command = append(command, "--no-setup", "--no-teardown", "--linger")
	}

	// For PLZ tests, we add a reserved env var containing instance ID.
	if len(tr.TestRunID()) > 0 && v1alpha1.IsTrue(tr, v1alpha1.CloudPLZTestRun) {
		command = append(command, "-e", fmt.Sprintf(`%s=%d`, cloud.IIDCloudExecVar, index))
	}

	command = script.UpdateCommand(command)

	var (
		zero32        int32 = 0
		schedulerName       = corev1.DefaultSchedulerName
	)

	image := "grafana/k6:latest"
	if tr.GetSpec().Runner.Image != "" {
		image = tr.GetSpec().Runner.Image
	}

	runnerAnnotations := make(map[string]string)
	if tr.GetSpec().Runner.Metadata.Annotations != nil {
		runnerAnnotations = tr.GetSpec().Runner.Metadata.Annotations
	}

	runnerLabels := newLabels(tr.NamespacedName().Name)
	runnerLabels["runner"] = "true"
	if tr.GetSpec().Runner.Metadata.Labels != nil {
		for k, v := range tr.GetSpec().Runner.Metadata.Labels { // Order not specified
			if _, ok := runnerLabels[k]; !ok {
				runnerLabels[k] = v
			}
		}
	}

	serviceAccountName := "default"
	if tr.GetSpec().Runner.ServiceAccountName != "" {
		serviceAccountName = tr.GetSpec().Runner.ServiceAccountName
	}

	automountServiceAccountToken := true
	if tr.GetSpec().Runner.AutomountServiceAccountToken != "" {
		automountServiceAccountToken, _ = strconv.ParseBool(tr.GetSpec().Runner.AutomountServiceAccountToken)
	}

	ports := []corev1.ContainerPort{{ContainerPort: 6565}}
	ports = append(ports, tr.GetSpec().Ports...)

	env := newIstioEnvVar(tr.GetSpec().Scuttle, istioEnabled)

	cloudEnvVars, err := getCloudEnvVars(tr, sti)
	if err != nil {
		return nil, err
	}
	env = append(env, cloudEnvVars...)

	env = append(env, tr.GetSpec().Runner.Env...)

	volumes := script.Volume()
	volumes = append(volumes, tr.GetSpec().Runner.Volumes...)

	volumeMounts := script.VolumeMount()
	volumeMounts = append(volumeMounts, tr.GetSpec().Runner.VolumeMounts...)

	if tr.GetSpec().Runner.SchedulerName != "" {
		schedulerName = tr.GetSpec().Runner.SchedulerName
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   tr.NamespacedName().Namespace,
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
					SchedulerName:                schedulerName,
					Affinity:                     tr.GetSpec().Runner.Affinity,
					NodeSelector:                 tr.GetSpec().Runner.NodeSelector,
					Tolerations:                  tr.GetSpec().Runner.Tolerations,
					TopologySpreadConstraints:    tr.GetSpec().Runner.TopologySpreadConstraints,
					SecurityContext:              &tr.GetSpec().Runner.SecurityContext,
					ImagePullSecrets:             tr.GetSpec().Runner.ImagePullSecrets,
					InitContainers:               getInitContainers(&tr.GetSpec().Runner, script),
					Containers: []corev1.Container{{
						Image:           image,
						ImagePullPolicy: tr.GetSpec().Runner.ImagePullPolicy,
						Name:            "k6",
						Command:         command,
						Env:             env,
						Resources:       tr.GetSpec().Runner.Resources,
						VolumeMounts:    volumeMounts,
						Ports:           ports,
						EnvFrom:         tr.GetSpec().Runner.EnvFrom,
						LivenessProbe:   generateProbe(tr.GetSpec().Runner.LivenessProbe),
						ReadinessProbe:  generateProbe(tr.GetSpec().Runner.ReadinessProbe),
						SecurityContext: &tr.GetSpec().Runner.ContainerSecurityContext,
					}},
					Volumes:           volumes,
					PriorityClassName: tr.GetSpec().Runner.PriorityClassName,
				},
			},
		},
	}

	if tr.GetSpec().Separate {
		job.Spec.Template.Spec.Affinity = newAntiAffinity()
	}

	return job, nil
}

func NewRunnerService(tr *v1alpha1.TestRun, index int) (*corev1.Service, error) {
	serviceName := fmt.Sprintf("%s-%s-%d", tr.NamespacedName().Name, "service", index)
	runnerName := fmt.Sprintf("%s-%d", tr.NamespacedName().Name, index)

	runnerAnnotations := make(map[string]string)
	if tr.GetSpec().Runner.Metadata.Annotations != nil {
		runnerAnnotations = tr.GetSpec().Runner.Metadata.Annotations
	}

	runnerLabels := newLabels(tr.NamespacedName().Name)
	runnerLabels["runner"] = "true"
	if tr.GetSpec().Runner.Metadata.Labels != nil {
		for k, v := range tr.GetSpec().Runner.Metadata.Labels { // Order not specified
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
			Namespace:   tr.NamespacedName().Namespace,
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
							{
								Key:      "runner",
								Operator: "In",
								Values: []string{
									"true",
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

func generateProbe(configuredProbe *corev1.Probe) *corev1.Probe {
	if configuredProbe != nil {
		return configuredProbe
	}
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/v1/status",
				Port:   intstr.IntOrString{IntVal: 6565},
				Scheme: "HTTP",
			},
		},
	}
}

// TODO: these are to be moved to an internal API package once it's up (separate PR & work, see #616)
func getCloudEnvVars(tr *v1alpha1.TestRun, sti *cloud.SecretTokenInfo) ([]corev1.EnvVar, error) {
	ev := make([]corev1.EnvVar, 0)

	// non-cloud mode - no-op
	if len(tr.TestRunID()) <= 0 {
		return ev, nil
	}

	ev = append(ev, tokenEnvVar(tr, sti)...)

	// Add aggregation vars for cloud output mode;
	// PLZ mode already has them.
	if !v1alpha1.IsTrue(tr, v1alpha1.CloudPLZTestRun) {
		aggregationVars, err := cloud.DecodeAggregationConfig(tr.GetStatus().AggregationVars)
		if err != nil {
			return nil, err
		}
		ev = append(ev, aggregationVars...)
	}

	ev = append(ev, corev1.EnvVar{
		Name:  "K6_CLOUD_PUSH_REF_ID",
		Value: tr.TestRunID(),
	})

	return ev, nil
}

// assumes tr is a cloud test run
func tokenEnvVar(tr *v1alpha1.TestRun, sti *cloud.SecretTokenInfo) []corev1.EnvVar {
	// cloud output mode
	// old auth path, with `--out cloud`
	if !v1alpha1.IsTrue(tr, v1alpha1.CloudPLZTestRun) {
		return []corev1.EnvVar{{
			Name:  "K6_CLOUD_TOKEN",
			Value: sti.Value(),
		}}
	}

	return nil

	// PLZ mode is a no-op because ephemeral `K6_CLOUD_TOKEN` is set in test run data

	// new auth path, with `--local-execution`
	//   K6_CLOUD_TEST_RUN_TOKEN for all modes
	// Future TODO, see https://github.com/grafana/k6-operator/issues/668
}
