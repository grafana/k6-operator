package jobs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/grafana/k6-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const initializerMissingK6Message = `level=error msg="k6 executable not found in PATH; initializer image must contain k6"`

// NewInitializerJob builds a template used to create an initializer job
func NewInitializerJob(k6 *v1alpha1.TestRun, archiveArgs []string) (*batchv1.Job, error) {
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

	var (
		// k6 allows to run archive command on archives too so type of file here doesn't matter
		scriptName  = script.FullName()
		archiveName = fmt.Sprintf("/tmp/%s.archived.tar", script.Filename)
	)
	istioCommand, istioEnabled := newIstioCommand(k6.GetSpec().Scuttle.Enabled, []string{"sh", "-c"})

	// NOTE: only .env are passed to k6 CLI, not .envFrom.
	// This is esp. relevant for the cloud output test where
	// duration of the test may depend on env var values.
	command := append(istioCommand,
		newInitializerCommand(scriptName, archiveName, k6.GetSpec().Initializer.Env, archiveArgs, k6.GetSpec().NeedsShellCmd())...)

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

// initializerShellScript renders the initializer's shell program.
// Parameters:
//   - setup: executed before k6 invocation; used by .spec.args flow.
//   - archiveRef: where k6 archive is stored.
//   - archiveCmd: the `k6 archive` command.
//
// About the log handling: there can be several scenarios from the k6 command:
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
func initializerShellScript(setup, archiveRef, archiveCmd string) string {
	if setup != "" {
		setup += "\n"
	}
	return fmt.Sprintf(
		`if ! command -v k6 >/dev/null 2>&1; then
  echo '%[1]s' >&2
  exit 127
fi

logs=/tmp/k6logs
%[2]s
if ! mkdir -p "$(dirname %[3]s)"; then
  exit 1
fi

if ! %[4]s 2> "${logs}"; then
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
		setup,
		archiveRef,
		archiveCmd,
	)
}

// also referenced by the unit tests
var initializerScript = initializerShellScript("archive=\"$1\"\nshift", `"${archive}"`, `"$@"`)

// newInitializerCommand builds the `sh -c` arguments that run the initializer.
// As the necessary step, it builds the archive command from the arguments.
// The command can take two forms:
//   - with .spec.arguments, via shell interpreter `sh -c` and concat of arguments.
//     The env var values are passed as ${..} to be expanded by Shell.
//   - with .spec.args, via shell positional arguments (more correct and safer)
//     The env var values are passed as $(..) to be expanded by kubelet.
//
// Env vars are passed as a ref by name to account for both Value and ValueFrom cases.
// The function assumes that the env vars are being passed to the container.
func newInitializerCommand(scriptName, archiveName string, env []corev1.EnvVar, archiveArgs []string, needsShellCmd bool) []string {
	if needsShellCmd {
		// .spec.arguments case
		var envVarString string
		for _, ev := range env {
			// Env values are expanded by the shell inside quotes.
			envVarString += fmt.Sprintf(` -e %s="${%s}"`, ev.Name, ev.Name)
		}

		// Arguments are joined back verbatim, retaining their original quote
		// context; `$(NAME)` refs in them are left for kubelet expansion.
		argLine := strings.Join(archiveArgs, " ")

		archiveCmd := fmt.Sprintf("k6 archive %s%s -O %s %s", scriptName, envVarString, archiveName, argLine)
		return []string{initializerShellScript("", archiveName, archiveCmd)}
	}

	// .spec.args case
	archiveCommand := []string{"k6", "archive", scriptName}
	for _, ev := range env {
		archiveCommand = append(archiveCommand, "-e", fmt.Sprintf("%s=$(%s)", ev.Name, ev.Name))
	}
	archiveCommand = append(archiveCommand, "-O", archiveName)
	archiveCommand = append(archiveCommand, archiveArgs...)

	// `sh` is a placeholder for $0 (unused in the script), archive name is $1.
	command := []string{initializerScript, "sh", archiveName}
	return append(command, archiveCommand...)
}
