package jobs

import (
	"fmt"
	"strings"
	"testing"

	deep "github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func defaultTestRunForInitializer() *v1alpha1.TestRun {
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
}

func defaultExpectedJobForInitializer() *batchv1.Job {
	script := &types.Script{
		Name:     "test",
		Filename: "test.js",
		Type:     "ConfigMap",
	}

	automountServiceAccountToken := true
	zero := int32(0)

	return &batchv1.Job{
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
					SchedulerName:                "default-scheduler",
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
							Command: append([]string{"sh", "-c"},
								newInitializerCommand(
									"/test/test.js",
									"/tmp/test.js.archived.tar",
									nil,
									[]string{"--out", "cloud"},
									true,
								)...),
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
}

func Test_NewInitializerJob(t *testing.T) {
	tests := []struct {
		name             string
		archiveArgs      []string
		setupTestRun     func(*v1alpha1.TestRun)
		setupExpectedJob func(*batchv1.Job)
	}{
		{
			name:             "base",
			archiveArgs:      []string{"--out", "cloud"},
			setupTestRun:     func(k6 *v1alpha1.TestRun) {},
			setupExpectedJob: func(j *batchv1.Job) {},
		},
		{
			name:        "with custom scheduler name",
			archiveArgs: []string{"--out", "cloud"},
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Initializer.SchedulerName = "custom-scheduler"
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.SchedulerName = "custom-scheduler"
			},
		},
		{
			name:        ".spec.args use the static script",
			archiveArgs: []string{"--tag", "note=hello world"},
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Arguments = ""
				k6.Spec.Args = []string{"--out", "cloud", "--tag", "note=hello world"}
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].Command = []string{
					"sh", "-c", initializerScript,
					"sh", "/tmp/test.js.archived.tar",
					"k6", "archive", "/test/test.js",
					"-O", "/tmp/test.js.archived.tar",
					"--tag", "note=hello world",
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			k6 := defaultTestRunForInitializer()

			if tt.setupTestRun != nil {
				tt.setupTestRun(k6)
			}

			expectedJob := defaultExpectedJobForInitializer()

			if tt.setupExpectedJob != nil {
				tt.setupExpectedJob(expectedJob)
			}

			job, err := NewInitializerJob(k6, tt.archiveArgs)

			if err != nil {
				t.Fatalf("NewInitializerJob errored: %v", err)
			}

			diff := deep.Equal(job, expectedJob)

			if diff != nil {
				t.Errorf("NewInitializerJob difference: %v", diff)
			}
		})
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
			expectedInEnvVar: []string{"FOO"},
		},
		{
			name:    "no env vars",
			setup:   func(k6 *v1alpha1.TestRun) {},
			noEFlag: true,
		},
		{
			name: "env vars with spaces",
			setup: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner = v1alpha1.Pod{
					Env: []corev1.EnvVar{
						{Name: "TEST_TAG", Value: "2026-03-31 TS Dev Smoke 212"},
					},
				}
			},
			expectedInEnvVar: []string{"TEST_TAG"},
		},
		{
			name: "env vars with special chars",
			setup: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner = v1alpha1.Pod{
					Env: []corev1.EnvVar{
						{Name: "THRESHOLDS", Value: "p(95),avg"},
					},
				}
			},
			expectedInEnvVar: []string{"THRESHOLDS"},
		},
		{
			name: "env var from a Secret",
			setup: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner = v1alpha1.Pod{
					Env: []corev1.EnvVar{
						{Name: "SECRET_TOKEN", ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "creds"},
								Key:                  "token",
							},
						}},
					},
				}
			},
			expectedInEnvVar: []string{"SECRET_TOKEN"},
		},
	}

	// modes creates a "matrix" of passing the same test cases to both
	// .spec.arguments and .spec.args. The diff is in format of env var.
	modes := []struct {
		name      string
		args      []string // add this to TestRun .spec.args
		envVarFmt string   // expected -e format
	}{
		{".spec.arguments", nil, `-e %s="${%s}"`},
		{".spec.args", []string{"--vus", "1"}, `-e %s=$(%s)`},
	}

	for _, mode := range modes {
		for _, tt := range tests {
			t.Run(mode.name+": "+tt.name, func(t *testing.T) {
				k6 := baseTestRun()
				tt.setup(k6)
				k6.Spec.Args = mode.args

				job, err := NewInitializerJob(k6, nil)
				if err != nil {
					t.Fatalf("NewInitializerJob errored: %v", err)
				}

				// find expected env vars in command and container's env vars
				cmd := strings.Join(job.Spec.Template.Spec.Containers[0].Command, " ")
				envVars := job.Spec.Template.Spec.Containers[0].Env

				for _, name := range tt.expectedInEnvVar {
					if want := fmt.Sprintf(mode.envVarFmt, name, name); !strings.Contains(cmd, want) {
						t.Errorf("command should contain %q, got: %s", want, cmd)
					}

					found := false
					for _, ev := range envVars {
						if ev.Name == name {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("container env should contain %q, got: %v", name, envVars)
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
}

// When YAML block scalar is used for .spec.arguments, it adds a newline to it.
// It can break the initializer shell script, so checking its removal.
func Test_InitializerCommand_ArgumentsRemoveNewlineInYamlBlock(t *testing.T) {
	k6 := &v1alpha1.TestRun{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: v1alpha1.TestRunSpec{
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{Name: "test", File: "test.js"},
			},
			Arguments: "--vus 10 --duration 5s\n",
		},
	}

	cli, err := types.ParseCLI(k6.Spec.Argv())
	if err != nil {
		t.Fatalf("ParseCLI errored: %v", err)
	}

	job, err := NewInitializerJob(k6, cli.ArchiveArgs)
	if err != nil {
		t.Fatalf("NewInitializerJob errored: %v", err)
	}

	shellSource := job.Spec.Template.Spec.Containers[0].Command[2]
	if want := `--duration 5s 2> "${logs}"`; !strings.Contains(shellSource, want) {
		t.Errorf("archive line should be one line ending in %q, got:\n%s", want, shellSource)
	}
}

func Test_InitializerScuttle(t *testing.T) {
	baseTestRun := func() *v1alpha1.TestRun {
		return &v1alpha1.TestRun{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
			Spec: v1alpha1.TestRunSpec{
				Script: v1alpha1.K6Script{
					ConfigMap: v1alpha1.K6Configmap{Name: "test", File: "test.js"},
				},
				Scuttle: v1alpha1.K6Scuttle{Enabled: "true"},
			},
		}
	}

	t.Run(".spec.arguments", func(t *testing.T) {
		k6 := baseTestRun()
		k6.Spec.Arguments = "--vus 1"

		job, err := NewInitializerJob(k6, []string{"--vus", "1"})
		if err != nil {
			t.Fatalf("NewInitializerJob errored: %v", err)
		}

		cmd := job.Spec.Template.Spec.Containers[0].Command
		if diff := deep.Equal(cmd, append([]string{"scuttle", "sh", "-c"},
			newInitializerCommand("/test/test.js", "/tmp/test.js.archived.tar", nil, []string{"--vus", "1"}, true)...)); diff != nil {
			t.Errorf("initializer command diff: %v", diff)
		}
	})

	t.Run(".spec.args", func(t *testing.T) {
		k6 := baseTestRun()
		k6.Spec.Args = []string{"--vus", "1"}

		job, err := NewInitializerJob(k6, []string{"--vus", "1"})
		if err != nil {
			t.Fatalf("NewInitializerJob errored: %v", err)
		}

		cmd := job.Spec.Template.Spec.Containers[0].Command
		if diff := deep.Equal(cmd, []string{
			"scuttle", "sh", "-c", initializerScript,
			"sh", "/tmp/test.js.archived.tar",
			"k6", "archive", "/test/test.js",
			"-O", "/tmp/test.js.archived.tar",
			"--vus", "1",
		}); diff != nil {
			t.Errorf("initializer command diff: %v", diff)
		}
	})
}

func Test_InitializerScript_NoDynamicValues(t *testing.T) {
	k6 := &v1alpha1.TestRun{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: v1alpha1.TestRunSpec{
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{Name: "test", File: "test.js"},
			},
			Args: []string{"--tag", "note=hello world; rm -rf /"}, // shell injection
		},
	}

	job, err := NewInitializerJob(k6, []string{"--tag", "note=hello world; rm -rf /"})
	if err != nil {
		t.Fatalf("NewInitializerJob errored: %v", err)
	}

	command := job.Spec.Template.Spec.Containers[0].Command
	if command[2] != initializerScript {
		t.Errorf("shell source is not the static script: %s", command[2])
	}
	// at this point, we can be certain that command[2] does not contain rm -rf
}
