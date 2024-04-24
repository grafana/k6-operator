package jobs

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/k6-operator/pkg/types"
)

type job struct {
	*batchv1.Job
}

func Job() *job {
	var zero32 int32
	return &job{
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "default",
				// Labels:      labels,
				// Annotations: annotations,
			},
			Spec: batchv1.JobSpec{
				BackoffLimit: &zero32,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						// Labels:      labels,
						// Annotations: annotations,
					},
					Spec: corev1.PodSpec{
						// AutomountServiceAccountToken: &automountServiceAccountToken,
						// ServiceAccountName:           serviceAccountName,
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{
							{
								Image: "ghcr.io/grafana/k6-operator:latest-runner",
								Name:  "k6",
								// Command:         command,
								// Env:             env,
								VolumeMounts: []corev1.VolumeMount{},
								Ports:        []corev1.ContainerPort{{ContainerPort: 6565}},
							},
						},
						Volumes: []corev1.Volume{},
					},
				},
			},
		}}
}

// mimic k8s pod for unit testing
func (j *job) GetPod() *corev1.Pod {
	labels := j.ObjectMeta.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	labels["job-name"] = j.ObjectMeta.Name
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        j.ObjectMeta.Name,
			Namespace:   j.ObjectMeta.Namespace,
			Labels:      labels,
			Annotations: j.ObjectMeta.Annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				j.Spec.Template.Spec.Containers[0],
			},
			Volumes: j.Spec.Template.Spec.Volumes,
		},
	}
}

func (j *job) Name(n string) *job {
	labels := newLabels(n)
	j.ObjectMeta.Labels = labels
	j.Spec.Template.ObjectMeta.Labels = labels
	j.ObjectMeta.Name = n
	return j
}

func (j *job) Namespace(ns string) *job {
	j.ObjectMeta.Namespace = ns
	return j
}

func (j *job) Script(script *types.Script) *job {
	volumes := script.Volume()
	j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, volumes...)

	volumeMounts := script.VolumeMount()
	j.Spec.Template.Spec.Containers[0].VolumeMounts = append(j.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMounts...)
	return j
}

func (j *job) Initializer(script *types.Script) *job {
	j.ObjectMeta.Name = j.ObjectMeta.Name + "-initializer"

	j.Script(script)

	// k6 allows to run archive command on archives too so type of file here doesn't matter
	scriptName := script.FullName()
	archiveName := fmt.Sprintf("/tmp/%s.archived.tar", script.Filename)

	// TODO add istio, refactor initializer job definition
	var argLine string // TODO
	j.Spec.Template.Spec.Containers[0].Command = []string{fmt.Sprintf(
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
		archiveName)}
	return j
}
