package jobs

import (
	"fmt"
	"strconv"

	"github.com/grafana/k6-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewInitializerJob builds a template used to initializefor creating a starter job
func NewInitializerJob(k6 *v1alpha1.K6, argLine string) (*batchv1.Job, error) {
	script, err := k6.Spec.ParseScript()
	if err != nil {
		return nil, err
	}

	var (
		image                        = "ghcr.io/grafana/k6-operator:latest-runner"
		annotations                  = make(map[string]string)
		labels                       = newLabels(k6.Name)
		serviceAccountName           = "default"
		automountServiceAccountToken = true
		ports                        = append([]corev1.ContainerPort{{ContainerPort: 6565}}, k6.Spec.Ports...)
	)

	if k6.Spec.Initializer == nil {
		k6.Spec.Initializer = k6.Spec.Runner.DeepCopy()
	}

	if k6.Spec.Initializer.Image != "" {
		image = k6.Spec.Initializer.Image
	}

	if k6.Spec.Initializer.Metadata.Annotations != nil {
		annotations = k6.Spec.Initializer.Metadata.Annotations
	}

	if k6.Spec.Initializer.Metadata.Labels != nil {
		for k, v := range k6.Spec.Initializer.Metadata.Labels {
			if _, ok := labels[k]; !ok {
				labels[k] = v
			}
		}
	}

	if k6.Spec.Initializer.ServiceAccountName != "" {
		serviceAccountName = k6.Spec.Initializer.ServiceAccountName
	}

	if k6.Spec.Initializer.AutomountServiceAccountToken != "" {
		automountServiceAccountToken, _ = strconv.ParseBool(k6.Spec.Initializer.AutomountServiceAccountToken)
	}

	var (
		// k6 allows to run archive command on archives too so type of file here doesn't matter
		scriptName  = script.FullName()
		archiveName = fmt.Sprintf("/tmp/%s.archived.tar", script.Filename)
	)
	istioCommand, istioEnabled := newIstioCommand(k6.Spec.Scuttle.Enabled, []string{"sh", "-c"})
	command := append(istioCommand, fmt.Sprintf(
		// There can be several scenarios from k6 command here:
		// a) script is correct and `k6 inspect` outputs JSON
		// b) script is partially incorrect and `k6` outputs a warning log message and
		// then a JSON
		// c) script is incorrect and `k6` outputs an error log message
		// Warnings at this point are not necessary (warning messages will re-appear in
		// runner's logs and the user can see them there) so we need a pure JSON here
		// without any additional messages in cases a) and b). In case c), output should
		// contain error message and the Job is to exit with non-zero code.
		//
		// Due to some pecularities of k6 logging, to achieve the above behaviour,
		// we need to use a workaround to store all log messages in temp file while
		// printing JSON as usual. Then parse temp file only for errors, ignoring
		// any other log messages.
		// Related: https://github.com/grafana/k6-docs/issues/877
		"mkdir -p $(dirname %s) && k6 archive %s -O %s %s 2> /tmp/k6logs && k6 inspect --execution-requirements %s 2> /tmp/k6logs ; ! cat /tmp/k6logs | grep 'level=error'",
		archiveName, scriptName, archiveName, argLine,
		archiveName))

	env := append(newIstioEnvVar(k6.Spec.Scuttle, istioEnabled), k6.Spec.Initializer.Env...)

	volumes := script.Volume()
	volumes = append(volumes, k6.Spec.Initializer.Volumes...)

	volumeMounts := script.VolumeMount()
	volumeMounts = append(volumeMounts, k6.Spec.Initializer.VolumeMounts...)

	var zero32 int32
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-initializer", k6.Name),
			Namespace:   k6.Namespace,
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
					Affinity:                     k6.Spec.Initializer.Affinity,
					NodeSelector:                 k6.Spec.Initializer.NodeSelector,
					Tolerations:                  k6.Spec.Initializer.Tolerations,
					SecurityContext:              &k6.Spec.Initializer.SecurityContext,
					RestartPolicy:                corev1.RestartPolicyNever,
					ImagePullSecrets:             k6.Spec.Initializer.ImagePullSecrets,
					InitContainers:               getInitContainers(&k6.Spec, script),
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: k6.Spec.Initializer.ImagePullPolicy,
							Name:            "k6",
							Command:         command,
							Env:             env,
							Resources:       k6.Spec.Initializer.Resources,
							VolumeMounts:    volumeMounts,
							Ports:           ports,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return job, nil
}
