package jobs

import (
	"errors"
	"reflect"
	"testing"

	deep "github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewScriptVolumeClaim(t *testing.T) {
	expectedOutcome := &types.Script{
		Name:     "Test",
		Path:     "/test/",
		Filename: "thing.js",
		Type:     "VolumeClaim",
	}

	k6 := v1alpha1.K6Spec{
		Script: v1alpha1.K6Script{
			VolumeClaim: v1alpha1.K6VolumeClaim{
				Name: "Test",
				File: "thing.js",
			},
		},
	}

	script, err := types.ParseScript(&k6)
	if err != nil {
		t.Errorf("NewScript with ConfigMap errored, got: %v, want: %v", err, expectedOutcome)
	}
	if !reflect.DeepEqual(script, expectedOutcome) {
		t.Errorf("NewScript with VolumeClaim failed to return expected output, got: %v, expected: %v", script, expectedOutcome)
	}
}

func TestNewScriptConfigMap(t *testing.T) {
	expectedOutcome := &types.Script{
		Name:     "Test",
		Path:     "/test/",
		Filename: "thing.js",
		Type:     "ConfigMap",
	}

	k6 := v1alpha1.K6Spec{
		Script: v1alpha1.K6Script{
			ConfigMap: v1alpha1.K6Configmap{
				Name: "Test",
				File: "thing.js",
			},
		},
	}

	script, err := types.ParseScript(&k6)
	if err != nil {
		t.Errorf("NewScript with ConfigMap errored, got: %v, want: %v", err, expectedOutcome)
	}
	if !reflect.DeepEqual(script, expectedOutcome) {
		t.Errorf("NewScript with ConfigMap failed to return expected output, got: %v, expected: %v", script, expectedOutcome)
	}
}

func TestNewScriptLocalFile(t *testing.T) {

	expectedOutcome := &types.Script{
		Name:     "LocalFile",
		Path:     "/custom/",
		Filename: "my_test.js",
		Type:     "LocalFile",
	}

	k6 := v1alpha1.K6Spec{
		Script: v1alpha1.K6Script{
			LocalFile: "/custom/my_test.js",
		},
	}

	script, err := types.ParseScript(&k6)
	if err != nil {
		t.Errorf("NewScript with LocalFile errored, got: %v, want: %v", err, expectedOutcome)
	}
	if !reflect.DeepEqual(script, expectedOutcome) {
		t.Errorf("NewScript with LocalFile failed to return expected output, got: %v, expected: %v", script, expectedOutcome)
	}
}

func TestNewScriptNoScript(t *testing.T) {
	k6 := v1alpha1.K6Spec{}

	script, err := types.ParseScript(&k6)
	if err == nil && script != nil {
		t.Errorf("Expected Error from NewScript, got: %v, want: %v", err, errors.New("configMap, VolumeClaim or LocalFile not provided in script definition"))
	}
}

func TestNewVolumeSpecVolumeClaim(t *testing.T) {
	expectedOutcome := []corev1.Volume{
		corev1.Volume{
			Name: "k6-test-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "test",
				},
			},
		},
	}

	script := &types.Script{
		Type: "VolumeClaim",
		Name: "test",
	}

	volumeSpec := script.Volume()
	if !reflect.DeepEqual(volumeSpec, expectedOutcome) {
		t.Errorf("VolumeSpec wasn't as expected, got: %v, expected: %v", volumeSpec, expectedOutcome)
	}
}

func TestNewVolumeSpecConfigMap(t *testing.T) {
	expectedOutcome := []corev1.Volume{
		corev1.Volume{
			Name: "k6-test-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test",
					},
				},
			},
		},
	}

	script := &types.Script{
		Type: "ConfigMap",
		Name: "test",
	}

	volumeSpec := script.Volume()
	if !reflect.DeepEqual(volumeSpec, expectedOutcome) {
		t.Errorf("VolumeSpec wasn't as expected, got: %v, expected: %v", volumeSpec, expectedOutcome)
	}
}

func TestNewVolumeSpecNoType(t *testing.T) {
	expectedOutcome := []corev1.Volume{}

	script := &types.Script{
		Name: "test",
	}

	volumeSpec := script.Volume()
	if !reflect.DeepEqual(volumeSpec, expectedOutcome) {
		t.Errorf("VolumeSpec wasn't as expected, got: %v, expected: %v", volumeSpec, expectedOutcome)
	}

}

func TestNewAntiAffinity(t *testing.T) {
	expectedOutcome := &corev1.Affinity{
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

	antiAffinity := newAntiAffinity()

	if !reflect.DeepEqual(antiAffinity, expectedOutcome) {
		t.Errorf("AntiAffinity returning unexpected values, got: %v, expected: %v", antiAffinity, expectedOutcome)
	}
}

func TestNewRunnerService(t *testing.T) {
	expectedOutcome := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-1",
			Namespace: "test",
			Labels: map[string]string{
				"app":    "k6",
				"k6_cr":  "test",
				"runner": "true",
				"label1": "awesome",
			},
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			}},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:     "http-api",
				Port:     6565,
				Protocol: "TCP",
			}},
			Selector: map[string]string{
				"job-name": "test-1",
			},
		},
	}

	k6 := &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.K6Spec{
			Runner: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
			},
		},
	}

	service, err := NewRunnerService(k6, 1)
	if err != nil {
		t.Errorf("NewRunnerService with errored, got: %v, want: %v", err, expectedOutcome)
	}
	if diff := deep.Equal(service, expectedOutcome); diff != nil {
		t.Errorf("NewRunnerService returned unexpected data, diff: %s", diff)
	}
}

func TestNewRunnerJob(t *testing.T) {
	script := &types.Script{
		Name:     "test",
		Filename: "thing.js",
		Type:     "ConfigMap",
	}

	var zero int64 = 0
	automountServiceAccountToken := true

	expectedLabels := map[string]string{
		"app":    "k6",
		"k6_cr":  "test",
		"runner": "true",
		"label1": "awesome",
	}

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
			Labels:    expectedLabels,
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: expectedLabels,
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					Hostname:                     "test-1",
					RestartPolicy:                corev1.RestartPolicyNever,
					SecurityContext:              &corev1.PodSecurityContext{},
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					ServiceAccountName:           "default",
					AutomountServiceAccountToken: &automountServiceAccountToken,
					Containers: []corev1.Container{{
						Image:           "ghcr.io/grafana/operator:latest-runner",
						ImagePullPolicy: corev1.PullNever,
						Name:            "k6",
						Command:         []string{"k6", "run", "--quiet", "/test/test.js", "--address=0.0.0.0:6565", "--paused", "--tag", "instance_id=1", "--tag", "job_name=test-1"},
						Env:             []corev1.EnvVar{},
						Resources:       corev1.ResourceRequirements{},
						VolumeMounts:    script.VolumeMount(),
						Ports:           []corev1.ContainerPort{{ContainerPort: 6565}},
						EnvFrom: []corev1.EnvFromSource{
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "env",
									},
								},
							},
						},
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}
	k6 := &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.K6Spec{
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Runner: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				EnvFrom: []corev1.EnvFromSource{
					{
						ConfigMapRef: &corev1.ConfigMapEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "env",
							},
						},
					},
				},
				ImagePullPolicy: corev1.PullNever,
			},
		},
	}

	job, err := NewRunnerJob(k6, 1, "", "")
	if err != nil {
		t.Errorf("NewRunnerJob errored, got: %v", err)
	}
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Errorf("NewRunnerJob returned unexpected data, diff: %s", diff)
	}
}

func TestNewRunnerJobNoisy(t *testing.T) {
	script := &types.Script{
		Name:     "test",
		Filename: "thing.js",
		Type:     "ConfigMap",
	}

	var zero int64 = 0
	automountServiceAccountToken := true

	expectedLabels := map[string]string{
		"app":    "k6",
		"k6_cr":  "test",
		"runner": "true",
		"label1": "awesome",
	}

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
			Labels:    expectedLabels,
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: expectedLabels,
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					Hostname:                     "test-1",
					RestartPolicy:                corev1.RestartPolicyNever,
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					ServiceAccountName:           "default",
					AutomountServiceAccountToken: &automountServiceAccountToken,
					SecurityContext:              &corev1.PodSecurityContext{},
					Containers: []corev1.Container{{
						Image:           "ghcr.io/grafana/operator:latest-runner",
						ImagePullPolicy: "",
						Name:            "k6",
						Command:         []string{"k6", "run", "/test/test.js", "--address=0.0.0.0:6565", "--paused", "--tag", "instance_id=1", "--tag", "job_name=test-1"},
						Env:             []corev1.EnvVar{},
						Resources:       corev1.ResourceRequirements{},
						VolumeMounts:    script.VolumeMount(),
						Ports:           []corev1.ContainerPort{{ContainerPort: 6565}},
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}
	k6 := &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.K6Spec{
			Quiet: "false",
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Runner: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
			},
		},
	}

	job, err := NewRunnerJob(k6, 1, "", "")
	if err != nil {
		t.Errorf("NewRunnerJob errored, got: %v", err)
	}
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Errorf("NewRunnerJob returned unexpected data, diff: %s", diff)
	}
}

func TestNewRunnerJobUnpaused(t *testing.T) {
	script := &types.Script{
		Name:     "test",
		Filename: "thing.js",
		Type:     "ConfigMap",
	}

	var zero int64 = 0
	automountServiceAccountToken := true

	expectedLabels := map[string]string{
		"app":    "k6",
		"k6_cr":  "test",
		"runner": "true",
		"label1": "awesome",
	}

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
			Labels:    expectedLabels,
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: expectedLabels,
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					Hostname:                     "test-1",
					RestartPolicy:                corev1.RestartPolicyNever,
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					ServiceAccountName:           "default",
					AutomountServiceAccountToken: &automountServiceAccountToken,
					SecurityContext:              &corev1.PodSecurityContext{},
					Containers: []corev1.Container{{
						Image:           "ghcr.io/grafana/operator:latest-runner",
						ImagePullPolicy: "",
						Name:            "k6",
						Command:         []string{"k6", "run", "--quiet", "/test/test.js", "--address=0.0.0.0:6565", "--tag", "instance_id=1", "--tag", "job_name=test-1"},
						Env:             []corev1.EnvVar{},
						Resources:       corev1.ResourceRequirements{},
						VolumeMounts:    script.VolumeMount(),
						Ports:           []corev1.ContainerPort{{ContainerPort: 6565}},
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}
	k6 := &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.K6Spec{
			Paused: "false",
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Runner: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
			},
		},
	}

	job, err := NewRunnerJob(k6, 1, "", "")
	if err != nil {
		t.Errorf("NewRunnerJob errored, got: %v", err)
	}
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Errorf("NewRunnerJob returned unexpected data, diff: %s", diff)
	}
}

func TestNewRunnerJobArguments(t *testing.T) {
	script := &types.Script{
		Name:     "test",
		Filename: "thing.js",
		Type:     "ConfigMap",
	}

	var zero int64 = 0
	automountServiceAccountToken := true

	expectedLabels := map[string]string{
		"app":    "k6",
		"k6_cr":  "test",
		"runner": "true",
		"label1": "awesome",
	}

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
			Labels:    expectedLabels,
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: expectedLabels,
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					Hostname:                     "test-1",
					RestartPolicy:                corev1.RestartPolicyNever,
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					ServiceAccountName:           "default",
					AutomountServiceAccountToken: &automountServiceAccountToken,
					SecurityContext:              &corev1.PodSecurityContext{},
					Containers: []corev1.Container{{
						Image:           "ghcr.io/grafana/operator:latest-runner",
						ImagePullPolicy: "",
						Name:            "k6",
						Command:         []string{"k6", "run", "--quiet", "--cool-thing", "/test/test.js", "--address=0.0.0.0:6565", "--paused", "--tag", "instance_id=1", "--tag", "job_name=test-1"},
						Env:             []corev1.EnvVar{},
						Resources:       corev1.ResourceRequirements{},
						VolumeMounts:    script.VolumeMount(),
						Ports:           []corev1.ContainerPort{{ContainerPort: 6565}},
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}

	k6 := &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.K6Spec{
			Arguments: "--cool-thing",
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Runner: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
			},
		},
	}

	job, err := NewRunnerJob(k6, 1, "", "")
	if err != nil {
		t.Errorf("NewRunnerJob errored, got: %v", err)
	}
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Errorf("NewRunnerJob returned unexpected data, diff: %s", diff)
	}
}

func TestNewRunnerJobServiceAccount(t *testing.T) {
	script := &types.Script{
		Name:     "test",
		Filename: "thing.js",
		Type:     "ConfigMap",
	}

	var zero int64 = 0
	automountServiceAccountToken := true

	expectedLabels := map[string]string{
		"app":    "k6",
		"k6_cr":  "test",
		"runner": "true",
		"label1": "awesome",
	}

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
			Labels:    expectedLabels,
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: expectedLabels,
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					Hostname:                     "test-1",
					RestartPolicy:                corev1.RestartPolicyNever,
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					ServiceAccountName:           "test",
					AutomountServiceAccountToken: &automountServiceAccountToken,
					SecurityContext:              &corev1.PodSecurityContext{},
					Containers: []corev1.Container{{
						Image:           "ghcr.io/grafana/operator:latest-runner",
						ImagePullPolicy: "",
						Name:            "k6",
						Command:         []string{"k6", "run", "--quiet", "/test/test.js", "--address=0.0.0.0:6565", "--paused", "--tag", "instance_id=1", "--tag", "job_name=test-1"},
						Env:             []corev1.EnvVar{},
						Resources:       corev1.ResourceRequirements{},
						VolumeMounts:    script.VolumeMount(),
						Ports:           []corev1.ContainerPort{{ContainerPort: 6565}},
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}

	k6 := &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.K6Spec{

			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Runner: v1alpha1.Pod{
				ServiceAccountName: "test",
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
			},
		},
	}

	job, err := NewRunnerJob(k6, 1, "", "")
	if err != nil {
		t.Errorf("NewRunnerJob errored, got: %v", err)
	}
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Errorf("NewRunnerJob returned unexpected data, diff: %s", diff)
	}
}

func TestNewRunnerJobIstio(t *testing.T) {
	script := &types.Script{
		Name:     "test",
		Filename: "thing.js",
		Type:     "ConfigMap",
	}

	var zero int64 = 0
	automountServiceAccountToken := true

	expectedLabels := map[string]string{
		"app":    "k6",
		"k6_cr":  "test",
		"runner": "true",
		"label1": "awesome",
	}

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
			Labels:    expectedLabels,
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: expectedLabels,
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					Hostname:                     "test-1",
					RestartPolicy:                corev1.RestartPolicyNever,
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					ServiceAccountName:           "default",
					AutomountServiceAccountToken: &automountServiceAccountToken,
					SecurityContext:              &corev1.PodSecurityContext{},
					Containers: []corev1.Container{{
						Image:           "ghcr.io/grafana/operator:latest-runner",
						ImagePullPolicy: "",
						Name:            "k6",
						Command:         []string{"scuttle", "k6", "run", "--quiet", "/test/test.js", "--address=0.0.0.0:6565", "--paused", "--tag", "instance_id=1", "--tag", "job_name=test-1"},
						Env: []corev1.EnvVar{
							{
								Name:  "ENVOY_ADMIN_API",
								Value: "http://127.0.0.1:15000",
							},
							{
								Name:  "ISTIO_QUIT_API",
								Value: "http://127.0.0.1:15020",
							},
							{
								Name:  "WAIT_FOR_ENVOY_TIMEOUT",
								Value: "15",
							},
						},
						Resources:    corev1.ResourceRequirements{},
						VolumeMounts: script.VolumeMount(),
						Ports:        []corev1.ContainerPort{{ContainerPort: 6565}},
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}
	k6 := &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.K6Spec{
			Scuttle: v1alpha1.K6Scuttle{
				Enabled: "true",
			},
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Runner: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
			},
		},
	}

	job, err := NewRunnerJob(k6, 1, "", "")
	if err != nil {
		t.Errorf("NewRunnerJob errored, got: %v", err)
	}
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Errorf("NewRunnerJob returned unexpected data, diff: %s", diff)
	}
}

func TestNewRunnerJobCloud(t *testing.T) {
	script := &types.Script{
		Name:     "test",
		Filename: "thing.js",
		Type:     "ConfigMap",
	}

	var zero int64 = 0
	automountServiceAccountToken := true

	expectedLabels := map[string]string{
		"app":    "k6",
		"k6_cr":  "test",
		"runner": "true",
	}

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
			Labels:    expectedLabels,
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: expectedLabels,
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					Hostname:                     "test-1",
					RestartPolicy:                corev1.RestartPolicyNever,
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					ServiceAccountName:           "default",
					SecurityContext:              &corev1.PodSecurityContext{},
					AutomountServiceAccountToken: &automountServiceAccountToken,
					Containers: []corev1.Container{{
						Image:           "ghcr.io/grafana/operator:latest-runner",
						ImagePullPolicy: "",
						Name:            "k6",
						Command:         []string{"k6", "run", "--quiet", "--out", "cloud", "/test/test.js", "--address=0.0.0.0:6565", "--paused", "--tag", "instance_id=1", "--tag", "job_name=test-1"},
						Env: []corev1.EnvVar{
							{
								Name:  "K6_CLOUD_PUSH_REF_ID",
								Value: "testrunid",
							},
							{
								Name:  "K6_CLOUD_TOKEN",
								Value: "token",
							},
						},
						Resources:    corev1.ResourceRequirements{},
						VolumeMounts: script.VolumeMount(),
						Ports:        []corev1.ContainerPort{{ContainerPort: 6565}},
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}
	k6 := &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.K6Spec{
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Arguments: "--out cloud",
			Runner: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
			},
		},
	}

	job, err := NewRunnerJob(k6, 1, "testrunid", "token")
	if err != nil {
		t.Errorf("NewRunnerJob errored, got: %v", err)
	}
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Errorf("NewRunnerJob returned unexpected data, diff: %s", diff)
	}
}

func TestNewRunnerJobLocalFile(t *testing.T) {
	script := &types.Script{
		Name:     "test",
		Filename: "/test/test.js",
		Type:     "LocalFile",
	}

	var zero int64 = 0
	automountServiceAccountToken := true

	expectedLabels := map[string]string{
		"app":    "k6",
		"k6_cr":  "test",
		"runner": "true",
		"label1": "awesome",
	}
	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "test",
			Labels:    expectedLabels,
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: expectedLabels,
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					Hostname:                     "test-1",
					RestartPolicy:                corev1.RestartPolicyNever,
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					ServiceAccountName:           "default",
					AutomountServiceAccountToken: &automountServiceAccountToken,
					SecurityContext:              &corev1.PodSecurityContext{},
					Containers: []corev1.Container{{
						Image:           "ghcr.io/grafana/operator:latest-runner",
						ImagePullPolicy: "",
						Name:            "k6",
						Command:         []string{"sh", "-c", "if [ ! -f /test/test.js ]; then echo \"LocalFile not found exiting...\"; exit 1; fi;\nk6 run --quiet /test/test.js --address=0.0.0.0:6565 --paused --tag instance_id=1 --tag job_name=test-1"},
						Env:             []corev1.EnvVar{},
						Resources:       corev1.ResourceRequirements{},
						VolumeMounts:    script.VolumeMount(),
						Ports:           []corev1.ContainerPort{{ContainerPort: 6565}},
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}
	k6 := &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.K6Spec{
			Scuttle: v1alpha1.K6Scuttle{
				Enabled: "false",
			},
			Script: v1alpha1.K6Script{
				LocalFile: "/test/test.js",
			},
			Runner: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
			},
		},
	}

	job, err := NewRunnerJob(k6, 1, "", "")
	if err != nil {
		t.Errorf("NewRunnerJob errored, got: %v", err)
	}
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Errorf("NewRunnerJob returned unexpected data, diff: %s", diff)
	}
}
