package jobs

import (
	"fmt"
	"strconv"

	"github.com/grafana/k6-operator/pkg/types"

	"github.com/grafana/k6-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func newLabels(name string) map[string]string {
	return map[string]string{
		"app":   "k6",
		"k6_cr": name,
	}
}

func newIstioCommand(istioEnabled string, inheritedCommands []string) ([]string, bool) {
	istio := false
	if istioEnabled != "" {
		istio, _ = strconv.ParseBool(istioEnabled)
	}
	var command []string

	if istio {
		command = append(command, "scuttle")
	}

	command = append(command, inheritedCommands...)

	return command, istio
}

func newIstioEnvVar(istio v1alpha1.K6Scuttle, istioEnabled bool) []corev1.EnvVar {
	env := []corev1.EnvVar{}

	if istioEnabled {
		var istioQuitApi string
		var envoyAdminApi string
		var waitForEnvoyTimeout string

		if istio.EnvoyAdminApi != "" {
			envoyAdminApi = istio.EnvoyAdminApi
		} else {
			envoyAdminApi = "http://127.0.0.1:15000"
		}
		env = append(env, corev1.EnvVar{
			Name:  "ENVOY_ADMIN_API",
			Value: envoyAdminApi,
		})

		if istio.IstioQuitApi != "" {
			istioQuitApi = istio.IstioQuitApi
		} else {
			istioQuitApi = "http://127.0.0.1:15020"
		}
		env = append(env, corev1.EnvVar{
			Name:  "ISTIO_QUIT_API",
			Value: istioQuitApi,
		})

		if istio.WaitForEnvoyTimeout != "" {
			waitForEnvoyTimeout = istio.WaitForEnvoyTimeout
		} else {
			waitForEnvoyTimeout = "15"
		}
		env = append(env, corev1.EnvVar{
			Name:  "WAIT_FOR_ENVOY_TIMEOUT",
			Value: waitForEnvoyTimeout,
		})

		if istio.NeverKillIstio {
			env = append(env, corev1.EnvVar{
				Name:  "NEVER_KILL_ISTIO",
				Value: strconv.FormatBool(istio.NeverKillIstio),
			})
		}

		if istio.NeverKillIstioOnFailure {
			env = append(env, corev1.EnvVar{
				Name:  "NEVER_KILL_ISTIO_ON_FAILURE",
				Value: strconv.FormatBool(istio.NeverKillIstioOnFailure),
			})
		}

		if istio.DisableLogging {
			env = append(env, corev1.EnvVar{
				Name:  "SCUTTLE_LOGGING",
				Value: strconv.FormatBool(false),
			})
		}

		if istio.StartWithoutEnvoy {
			env = append(env, corev1.EnvVar{
				Name:  "START_WITHOUT_ENVOY",
				Value: strconv.FormatBool(istio.StartWithoutEnvoy),
			})
		}

		if istio.GenericQuitEndpoint != "" {
			env = append(env, corev1.EnvVar{
				Name:  "GENERIC_QUIT_ENDPOINT",
				Value: istio.GenericQuitEndpoint,
			})
		}

		if istio.QuitWithoutEnvoyTimeout != "" {
			env = append(env, corev1.EnvVar{
				Name:  "QUIT_WITHOUT_ENVOY_TIMEOUT",
				Value: istio.QuitWithoutEnvoyTimeout,
			})
		}
	}
	return env
}

// TODO: Envoy variables are not passed to init containers
func getInitContainers(pod *v1alpha1.Pod, script *types.Script) []corev1.Container {
	var initContainers []corev1.Container

	for i, k6InitContainer := range pod.InitContainers {

		name := fmt.Sprintf("k6-init-%d", i)
		if k6InitContainer.Name != "" {
			name = k6InitContainer.Name
		}

		volumeMounts := append(script.VolumeMount(), k6InitContainer.VolumeMounts...)

		initContainer := corev1.Container{
			Name:            name,
			Image:           k6InitContainer.Image,
			Command:         k6InitContainer.Command,
			Args:            k6InitContainer.Args,
			WorkingDir:      k6InitContainer.WorkingDir,
			EnvFrom:         k6InitContainer.EnvFrom,
			Env:             k6InitContainer.Env,
			VolumeMounts:    volumeMounts,
			ImagePullPolicy: pod.ImagePullPolicy,
			SecurityContext: &pod.ContainerSecurityContext,
		}
		initContainers = append(initContainers, initContainer)
	}

	return initContainers
}
