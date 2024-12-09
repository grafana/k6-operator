package jobs

import (
	"fmt"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/resources/containers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// NewStopJob builds a template used for creating a stop job
func NewStopJob(k6 *v1alpha1.TestRun, hostname []string) *batchv1.Job {
	// this job is almost identical to the starter so re-use the definitions
	job := NewStarterJob(k6, hostname)

	job.Name = fmt.Sprintf("%s-stopper", k6.NamespacedName().Name)

	image := "ghcr.io/grafana/k6-operator:latest-starter"
	if k6.GetSpec().Starter.Image != "" {
		image = k6.GetSpec().Starter.Image
	}

	command, istioEnabled := newIstioCommand(k6.GetSpec().Scuttle.Enabled, []string{"sh", "-c"})
	env := newIstioEnvVar(k6.GetSpec().Scuttle, istioEnabled)

	job.Spec.Template.Spec.Containers = []corev1.Container{
		containers.NewStopContainer(hostname, image, k6.GetSpec().Starter.ImagePullPolicy, command, env, k6.GetSpec().Starter.ContainerSecurityContext),
	}

	return job
}
