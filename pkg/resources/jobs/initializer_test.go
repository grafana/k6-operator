package jobs

import (
	"strings"
	"testing"

	deep "github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NewInitializerJob(t *testing.T) {
	script := &types.Script{
		Name:     "test",
		Filename: "test.js",
		Type:     "ConfigMap",
	}

	automountServiceAccountToken := true
	zero := int32(0)

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-initializer",
			Namespace: "test",
			Labels: map[string]string{
				"app":    "k6",
				"k6_cr":  "test",
				"label1": "awesome",
			},
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &zero,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":    "k6",
						"k6_cr":  "test",
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &automountServiceAccountToken,
					ServiceAccountName:           "default",
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					TopologySpreadConstraints:    nil,
					RestartPolicy:                corev1.RestartPolicyNever,
					SecurityContext:              &corev1.PodSecurityContext{},
					Containers: []corev1.Container{
						{
							Image:           "grafana/k6:latest",
							ImagePullPolicy: "",
							Name:            "k6",
							Command: []string{
								"sh", "-c",
								"mkdir -p $(dirname /tmp/test.js.archived.tar) && k6 archive /test/test.js -O /tmp/test.js.archived.tar --out cloud 2> /tmp/k6logs && k6 inspect --execution-requirements /tmp/test.js.archived.tar 2> /tmp/k6logs ; ! cat /tmp/k6logs | grep 'level=error'",
							},
							Env: []corev1.EnvVar{},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "env",
										},
									},
								},
							},
							Resources:       corev1.ResourceRequirements{},
							VolumeMounts:    script.VolumeMount(),
							Ports:           []corev1.ContainerPort{{ContainerPort: 6565}},
							SecurityContext: &corev1.SecurityContext{},
						},
					},
					Volumes: script.Volume(),
				},
			},
		},
	}

	k6 := &v1alpha1.TestRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.TestRunSpec{
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Arguments: "--out cloud",
			Initializer: &v1alpha1.Pod{
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
			},
		},
	}

	job, err := NewInitializerJob(k6, "--out cloud")
	if err != nil {
		t.Errorf("NewInitializerJob errored, got: %v", err)
	}

	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Error(diff)
	}
}

func Test_InitializerEnvVarFlags(t *testing.T) {
	baseTestRun := func() *v1alpha1.TestRun {
		return &v1alpha1.TestRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
			Spec: v1alpha1.TestRunSpec{
				Script: v1alpha1.K6Script{
					ConfigMap: v1alpha1.K6Configmap{
						Name: "test",
						File: "test.js",
					},
				},
			},
		}
	}

	tests := []struct {
		name             string
		setup            func(k6 *v1alpha1.TestRun)
		expectedInCmd    []string
		expectedInEnvVar []string
		noEFlag          bool
	}{
		{
			name: "env vars set in initializer",
			setup: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Initializer = &v1alpha1.Pod{
					Env: []corev1.EnvVar{
						{Name: "FOO", Value: "bar"},
						{Name: "OTHER", Value: "42"},
					},
				}
			},
			expectedInCmd:    []string{"-e FOO=bar", "-e OTHER=42"},
			expectedInEnvVar: []string{"FOO", "OTHER"},
		},
		{
			name: "env vars set only in runner",
			setup: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner = v1alpha1.Pod{
					Env: []corev1.EnvVar{
						{Name: "FOO", Value: "bar"},
					},
				}
			},
			expectedInCmd:    []string{"-e FOO=bar"},
			expectedInEnvVar: []string{"FOO"},
		},
		{
			name:    "no env vars",
			setup:   func(k6 *v1alpha1.TestRun) {},
			noEFlag: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k6 := baseTestRun()
			tt.setup(k6)

			job, err := NewInitializerJob(k6, "")
			if err != nil {
				t.Fatalf("NewInitializerJob errored: %v", err)
			}

			cmd := strings.Join(job.Spec.Template.Spec.Containers[0].Command, " ")

			for _, want := range tt.expectedInCmd {
				if !strings.Contains(cmd, want) {
					t.Errorf("command should contain %q, got: %s", want, cmd)
				}
			}

			envVars := job.Spec.Template.Spec.Containers[0].Env
			for _, expected := range tt.expectedInEnvVar {
				found := false
				for _, ev := range envVars {
					if ev.Name == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("container env should contain %q, got: %v", expected, envVars)
				}
			}

			if tt.noEFlag {
				if strings.Contains(cmd, " -e ") {
					t.Errorf("command should NOT contain `-e`, got: %s", cmd)
				}
			}
		})
	}
}
