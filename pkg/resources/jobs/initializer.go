package jobs

import (
	"fmt"
	"strconv"

	"github.com/grafana/k6-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const initializerMissingK6Message = `level=error msg="k6 executable not found in PATH; initializer image must contain k6"`

// NewInitializerJob builds a template used to initializefor creating a starter job
func NewInitializerJob(k6 *v1alpha1.TestRun, argLine string) (*batchv1.Job, error) {
	script, err := k6.GetSpec().ParseScript()
	if err != nil {
		return nil, err
	}

	var (
		image                        = "grafana/k6:latest"
		annotations                  = make(map[string]string)
		labels                       = newLabels(k6.NamespacedName().Name)
		serviceAccountName           = "default"
		automountServiceAccountToken = true
		ports                        = append([]corev1.ContainerPort{{ContainerPort: 6565}}, k6.GetSpec().Ports...)
		schedulerName                = corev1.DefaultSchedulerName
	)

	if k6.GetSpec().Initializer == nil {
		k6.GetSpec().Initializer = k6.GetSpec().Runner.DeepCopy()
	}

	if k6.GetSpec().Initializer.Image != "" {
		image = k6.GetSpec().Initializer.Image
	}

	if k6.GetSpec().Initializer.Metadata.Annotations != nil {
		annotations = k6.GetSpec().Initializer.Metadata.Annotations
	}

	if k6.GetSpec().Initializer.Metadata.Labels != nil {
		for k, v := range k6.GetSpec().Initializer.Metadata.Labels {
			if _, ok := labels[k]; !ok {
				labels[k] = v
			}
		}
	}

	if k6.GetSpec().Initializer.ServiceAccountName != "" {
		serviceAccountName = k6.GetSpec().Initializer.ServiceAccountName
	}

	if k6.GetSpec().Initializer.AutomountServiceAccountToken != "" {
		automountServiceAccountToken, _ = strconv.ParseBool(k6.GetSpec().Initializer.AutomountServiceAccountToken)
	}

	// NOTE: only .env are passed to k6 CLI, not .envFrom
	// This is esp. relevant for the cloud output test where
	// duration of the test may depend on env var values. IOW,
	// these env vars must always be passed in cloud output mode.
	var envVarString string
	for _, ev := range k6.GetSpec().Initializer.Env {
		envVarString += fmt.Sprintf(` -e %s="%s"`, ev.Name, ev.Value)
	}

	var (
		// k6 allows to run archive command on archives too so type of file here doesn't matter
		scriptName  = script.FullName()
		archiveName = fmt.Sprintf("/tmp/%s.archived.tar", script.Filename)
	)
	istioCommand, istioEnabled := newIstioCommand(k6.GetSpec().Scuttle.Enabled, []string{"sh", "-c"})
	command := append(istioCommand, newInitializerCommand(scriptName, archiveName, envVarString, argLine))

	env := append(newIstioEnvVar(k6.GetSpec().Scuttle, istioEnabled), k6.GetSpec().Initializer.Env...)

	volumes := script.Volume()
	volumes = append(volumes, k6.GetSpec().Initializer.Volumes...)

	volumeMounts := script.VolumeMount()
	volumeMounts = append(volumeMounts, k6.GetSpec().Initializer.VolumeMounts...)

	if k6.GetSpec().Initializer.SchedulerName != "" {
		schedulerName = k6.GetSpec().Initializer.SchedulerName
	}

	var zero32 int32
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-initializer", k6.NamespacedName().Name),
			Namespace:   k6.NamespacedName().Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &zero32,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &automountServiceAccountToken,
					ServiceAccountName:           serviceAccountName,
					Affinity:                     k6.GetSpec().Initializer.Affinity,
					NodeSelector:                 k6.GetSpec().Initializer.NodeSelector,
					Tolerations:                  k6.GetSpec().Initializer.Tolerations,
					TopologySpreadConstraints:    k6.GetSpec().Initializer.TopologySpreadConstraints,
					SecurityContext:              &k6.GetSpec().Initializer.SecurityContext,
					RestartPolicy:                corev1.RestartPolicyNever,
					SchedulerName:                schedulerName,
					ImagePullSecrets:             k6.GetSpec().Initializer.ImagePullSecrets,
					InitContainers:               getInitContainers(k6.GetSpec().Initializer, script),
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: k6.GetSpec().Initializer.ImagePullPolicy,
							Name:            "k6",
							Command:         command,
							Env:             env,
							Resources:       k6.GetSpec().Initializer.Resources,
							VolumeMounts:    volumeMounts,
							EnvFrom:         k6.GetSpec().Initializer.EnvFrom,
							Ports:           ports,
							SecurityContext: &k6.GetSpec().Initializer.ContainerSecurityContext,
						},
					},
					Volumes:           volumes,
					PriorityClassName: k6.GetSpec().Initializer.PriorityClassName,
				},
			},
		},
	}

	return job, nil
}

func newInitializerCommand(scriptName, archiveName, envVarString, argLine string) string {
	// There can be several scenarios from k6 command here:
	// a) script is correct and `k6 inspect` outputs JSON;
	// b) script is partially incorrect and `k6` outputs a warning log message and
	// then a JSON;
	// c) script is incorrect and `k6` outputs an error log message;
	// d) k6 binary is missing;
	// e) k6 binary exists but is corrupted or otherwise unexecutable.
	//
	// Warnings at this point are not necessary (warning messages will re-appear in
	// runner's logs and the user can see them there) so we need a pure JSON here,
	// without any additional messages in cases a) and b). In cases c) - e), output
	// should contain an error message and the Job is to exit with non-zero code.
	//
	// Due to some peculiarities of k6 logging, to achieve the above behaviour,
	// we need to use a workaround to store all log messages in temp file while
	// printing JSON as usual. Then parse temp file only for errors, ignoring
	// any other log messages.
	// Related: https://github.com/grafana/k6-docs/issues/877

	return fmt.Sprintf(
		`if ! command -v k6 >/dev/null 2>&1; then
  echo '%[1]s' >&2
  exit 127
fi

logs=/tmp/k6logs

if ! mkdir -p "$(dirname %[3]s)"; then
  exit 1
fi

if ! k6 archive %[2]s%[4]s -O %[3]s %[5]s 2> "${logs}"; then
  cat "${logs}"
  exit 1
fi

if ! k6 inspect --execution-requirements %[3]s 2> "${logs}"; then
  cat "${logs}"
  exit 1
fi

if grep 'level.*error' "${logs}"; then
  exit 1
fi`,
		initializerMissingK6Message,
		scriptName,
		archiveName,
		envVarString,
		argLine,
	)
}
