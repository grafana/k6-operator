package jobs

import (
	"fmt"
	"strconv"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewInitializerJob builds a template used to initializefor creating a starter job
func NewInitializerJob(k6 *v1alpha1.K6, argLine string) (*batchv1.Job, error) {
	script, err := types.ParseScript(&k6.Spec)
	if err != nil {
		return nil, err
	}

	var (
		image                        = "ghcr.io/grafana/operator:latest-runner"
		annotations                  = make(map[string]string)
		labels                       = newLabels(k6.Name)
		serviceAccountName           = "default"
		automountServiceAccountToken = true
		ports                        = append([]corev1.ContainerPort{{ContainerPort: 6565}}, k6.Spec.Ports...)
	)

	if k6.Spec.Runner.Image != "" {
		image = k6.Spec.Runner.Image
	}

	if k6.Spec.Runner.Metadata.Annotations != nil {
		annotations = k6.Spec.Runner.Metadata.Annotations
	}

	if k6.Spec.Runner.Metadata.Labels != nil {
		for k, v := range k6.Spec.Runner.Metadata.Labels { // Order not specified
			if _, ok := labels[k]; !ok {
				labels[k] = v
			}
		}
	}

	if k6.Spec.Runner.ServiceAccountName != "" {
		serviceAccountName = k6.Spec.Runner.ServiceAccountName
	}

	if k6.Spec.Runner.AutomountServiceAccountToken != "" {
		automountServiceAccountToken, _ = strconv.ParseBool(k6.Spec.Runner.AutomountServiceAccountToken)
	}

	var (
		// k6 allows to run archive command on archives too so type of file here doesn't matter
		scriptName  = script.FullName()
		archiveName = fmt.Sprintf("./%s.archived.tar", script.Filename)
	)
	command, istioEnabled := newIstioCommand(k6.Spec.Scuttle.Enabled, []string{"sh", "-c"})
	command = append(command, fmt.Sprintf(
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
		"k6 archive %s -O %s %s 2> /tmp/k6logs && k6 inspect --execution-requirements %s 2> /tmp/k6logs ; ! cat /tmp/k6logs | grep 'level=error'",
		scriptName, archiveName, argLine,
		archiveName))

	env := append(newIstioEnvVar(k6.Spec.Scuttle, istioEnabled), k6.Spec.Runner.Env...)

	var zero32 int32
	return &batchv1.Job{
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
					Affinity:                     k6.Spec.Runner.Affinity,
					NodeSelector:                 k6.Spec.Runner.NodeSelector,
					Tolerations:                  k6.Spec.Runner.Tolerations,
					RestartPolicy:                corev1.RestartPolicyNever,
					ImagePullSecrets:             k6.Spec.Runner.ImagePullSecrets,
					Containers: []corev1.Container{
						{
							Image:           image,
							ImagePullPolicy: k6.Spec.Runner.ImagePullPolicy,
							Name:            "k6",
							Command:         command,
							Env:             env,
							Resources:       k6.Spec.Runner.Resources,
							VolumeMounts:    script.VolumeMount(),
							Ports:           ports,
						},
					},
					Volumes: script.Volume(),
				},
			},
		},
	}, nil
}
