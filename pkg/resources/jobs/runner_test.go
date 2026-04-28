package jobs

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	deep "github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// these are default values hard-coded in k6
var aggregationEnvVars = []corev1.EnvVar{
	{
		Name:  "K6_CLOUD_API_VERSION",
		Value: "2",
	}, {
		Name:  "K6_CLOUD_AGGREGATION_PERIOD",
		Value: "5s",
	}, {
		Name:  "K6_CLOUD_AGGREGATION_WAIT_PERIOD",
		Value: "3s",
	}, {
		Name:  "K6_CLOUD_METRIC_PUSH_INTERVAL",
		Value: "10s",
	}, {
		Name:  "K6_CLOUD_METRIC_PUSH_CONCURRENCY",
		Value: "10",
	},
}

func defaultScript() *types.Script {
	return &types.Script{
		Name:     "test",
		Filename: "thing.js",
		Type:     "ConfigMap",
	}
}

func defaultLabels() map[string]string {
	return map[string]string{
		"app":    "k6",
		"k6_cr":  "test",
		"runner": "true",
	}
}

func customAnnotations() map[string]string {
	return map[string]string{
		"awesomeAnnotation": "dope",
	}
}

// Note: base TestRun in tests includes custom labels and annotations
func defaultTestRun() *v1alpha1.TestRun {
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
			Runner: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels:      map[string]string{"label1": "awesome"},
					Annotations: map[string]string{"awesomeAnnotation": "dope"},
				},
			},
		},
	}
}

func defaultExpectedJob(script *types.Script) *batchv1.Job {
	var zero int64 = 0
	automountServiceAccountToken := true

	jobLabels := defaultLabels()
	jobLabels["label1"] = "awesome"
	podLabels := defaultLabels()
	podLabels["label1"] = "awesome"

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-1",
			Namespace:   "test",
			Labels:      jobLabels,
			Annotations: customAnnotations(),
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: new(int32),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: customAnnotations(),
				},
				Spec: corev1.PodSpec{
					Hostname:                     "test-1",
					RestartPolicy:                corev1.RestartPolicyNever,
					SecurityContext:              &corev1.PodSecurityContext{},
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					TopologySpreadConstraints:    nil,
					ServiceAccountName:           "default",
					AutomountServiceAccountToken: &automountServiceAccountToken,
					Containers: []corev1.Container{{
						Image:           "grafana/k6:latest",
						ImagePullPolicy: "",
						Name:            "k6",
						Command:         []string{"k6", "run", "--quiet", "/test/test.js", "--address=0.0.0.0:6565", "--paused", "--tag", "instance_id=1", "--tag", "testrun_name=test"},
						Env:             []corev1.EnvVar{},
						Resources:       corev1.ResourceRequirements{},
						VolumeMounts:    script.VolumeMount(),
						Ports:           []corev1.ContainerPort{{ContainerPort: 6565}},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/v1/status",
									Port:   intstr.IntOrString{IntVal: 6565},
									Scheme: "HTTP",
								},
							},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/v1/status",
									Port:   intstr.IntOrString{IntVal: 6565},
									Scheme: "HTTP",
								},
							},
						},
						SecurityContext: &corev1.SecurityContext{},
					}},
					TerminationGracePeriodSeconds: &zero,
					Volumes:                       script.Volume(),
				},
			},
		},
	}
}

func envFromConfigMap(name string) []corev1.EnvFromSource {
	return []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		},
	}
}

func Test_NewRunnerJob(t *testing.T) {
	tests := []struct {
		name             string
		script           *types.Script
		tokenInfo        *cloud.TokenInfo
		setupTestRun     func(*v1alpha1.TestRun)
		setupExpectedJob func(*batchv1.Job)
	}{
		{
			name:             "base",
			setupTestRun:     func(k6 *v1alpha1.TestRun) {},
			setupExpectedJob: func(j *batchv1.Job) {},
		},
		{
			name: "passing basic fields",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner.ServiceAccountName = "test"
				k6.Spec.Runner.EnvFrom = envFromConfigMap("env")
				k6.Spec.Runner.ImagePullPolicy = corev1.PullNever
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.ServiceAccountName = "test"
				j.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullNever
				j.Spec.Template.Spec.Containers[0].EnvFrom = envFromConfigMap("env")
			},
		},
		{
			name: "modifying k6 command: noisy",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Quiet = "false"
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].Command = []string{
					"k6", "run", "/test/test.js", "--address=0.0.0.0:6565", "--paused",
					"--tag", "instance_id=1", "--tag", "testrun_name=test",
				}
			},
		},
		{
			name: "modifying k6 command: unpaused",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Paused = "false"
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].Command = []string{
					"k6", "run", "--quiet", "/test/test.js", "--address=0.0.0.0:6565",
					"--tag", "instance_id=1", "--tag", "testrun_name=test",
				}
			},
		},
		{
			name: "modifying k6 command: arguments",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Arguments = "--cool-thing"
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].Command = []string{
					"k6", "run", "--quiet", "--cool-thing", "/test/test.js", "--address=0.0.0.0:6565", "--paused",
					"--tag", "instance_id=1", "--tag", "testrun_name=test",
				}
			},
		},
		{
			name: "istio scuttle",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Scuttle = v1alpha1.K6Scuttle{Enabled: "true"}
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].Command = []string{
					"scuttle", "k6", "run", "--quiet", "/test/test.js", "--address=0.0.0.0:6565", "--paused",
					"--tag", "instance_id=1", "--tag", "testrun_name=test",
				}
				j.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
					{Name: "ENVOY_ADMIN_API", Value: "http://127.0.0.1:15000"},
					{Name: "ISTIO_QUIT_API", Value: "http://127.0.0.1:15020"},
					{Name: "WAIT_FOR_ENVOY_TIMEOUT", Value: "15"},
				}
			},
		},
		{
			name:      "cloud output mode",
			tokenInfo: cloud.NewTokenInfo("", "").InjectValue("token"),
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Arguments = "--out cloud"
				k6.Spec.Runner.Metadata.Labels = nil
				k6.Status.TestRunID = "testrunid"
				k6.Status.AggregationVars = "2|5s|3s|10s|10"
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Labels = defaultLabels()
				j.Spec.Template.Labels = defaultLabels()
				j.Spec.Template.Spec.Containers[0].Command = []string{
					"k6", "run", "--quiet", "--out", "cloud", "/test/test.js", "--address=0.0.0.0:6565", "--paused",
					"--tag", "instance_id=1", "--tag", "testrun_name=test",
				}
				j.Spec.Template.Spec.Containers[0].Env = append(
					[]corev1.EnvVar{{Name: "K6_CLOUD_TOKEN", Value: "token"}},
					append(aggregationEnvVars,
						corev1.EnvVar{Name: "K6_CLOUD_PUSH_REF_ID", Value: "testrunid"},
					)...,
				)
			},
		},
		{
			name: "LocalFile",
			script: &types.Script{
				Name:     "test",
				Filename: "/test/test.js",
				Type:     "LocalFile",
			},
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Script = v1alpha1.K6Script{LocalFile: "/test/test.js"}
				k6.Spec.Scuttle = v1alpha1.K6Scuttle{Enabled: "false"}
			},
			setupExpectedJob: func(j *batchv1.Job) {
				localScript := &types.Script{
					Name:     "test",
					Filename: "/test/test.js",
					Type:     "LocalFile",
				}
				j.Spec.Template.Spec.Containers[0].Command = []string{
					"sh", "-c",
					"if [ ! -f /test/test.js ]; then echo \"LocalFile not found exiting...\"; exit 1; fi;\nk6 run --quiet /test/test.js --address=0.0.0.0:6565 --paused --tag instance_id=1 --tag testrun_name=test",
				}
				j.Spec.Template.Spec.Containers[0].VolumeMounts = localScript.VolumeMount()
				j.Spec.Template.Spec.Volumes = localScript.Volume()
			},
		},
		{
			name: "init container",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner.EnvFrom = envFromConfigMap("env")
				k6.Spec.Runner.ImagePullPolicy = corev1.PullNever
				k6.Spec.Runner.InitContainers = []v1alpha1.InitContainer{
					{
						Image:   "busybox:1.28",
						Command: []string{"sh", "-c", "cat /test/test.js"},
						EnvFrom: envFromConfigMap("env"),
					},
				}
			},
			setupExpectedJob: func(j *batchv1.Job) {
				script := defaultScript()
				j.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullNever
				j.Spec.Template.Spec.Containers[0].EnvFrom = envFromConfigMap("env")
				j.Spec.Template.Spec.InitContainers = []corev1.Container{
					{
						Name:            "k6-init-0",
						Image:           "busybox:1.28",
						Command:         []string{"sh", "-c", "cat /test/test.js"},
						VolumeMounts:    script.VolumeMount(),
						ImagePullPolicy: corev1.PullNever,
						EnvFrom:         envFromConfigMap("env"),
						SecurityContext: &corev1.SecurityContext{},
					},
				}
			},
		},
		{
			name: "init container with volumes",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner.EnvFrom = envFromConfigMap("env")
				k6.Spec.Runner.ImagePullPolicy = corev1.PullNever
				extraMount := corev1.VolumeMount{Name: "k6-test-volume", MountPath: "/test/location"}
				k6.Spec.Runner.InitContainers = []v1alpha1.InitContainer{
					{
						Image:        "busybox:1.28",
						Command:      []string{"sh", "-c", "cat /test/test.js"},
						EnvFrom:      envFromConfigMap("env"),
						VolumeMounts: []corev1.VolumeMount{extraMount},
					},
				}
				k6.Spec.Runner.Volumes = []corev1.Volume{
					{
						Name:         "k6-test-volume",
						VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
					},
				}
				k6.Spec.Runner.VolumeMounts = []corev1.VolumeMount{extraMount}
			},
			setupExpectedJob: func(j *batchv1.Job) {
				script := defaultScript()
				extraMount := corev1.VolumeMount{Name: "k6-test-volume", MountPath: "/test/location"}
				expectedVolumeMounts := append(script.VolumeMount(), extraMount)

				j.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullNever
				j.Spec.Template.Spec.Containers[0].EnvFrom = envFromConfigMap("env")
				j.Spec.Template.Spec.Containers[0].VolumeMounts = expectedVolumeMounts
				j.Spec.Template.Spec.InitContainers = []corev1.Container{
					{
						Name:            "k6-init-0",
						Image:           "busybox:1.28",
						Command:         []string{"sh", "-c", "cat /test/test.js"},
						VolumeMounts:    expectedVolumeMounts,
						ImagePullPolicy: corev1.PullNever,
						EnvFrom:         envFromConfigMap("env"),
						SecurityContext: &corev1.SecurityContext{},
					},
				}
				j.Spec.Template.Spec.Volumes = append(script.Volume(), corev1.Volume{
					Name:         "k6-test-volume",
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				})
			},
		},
		{
			name: "priority class name",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner.PriorityClassName = "high-priority"
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.PriorityClassName = "high-priority"
			},
		},
		{
			name: "custom image",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner.Image = "grafana/k6:0.50.0"
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].Image = "grafana/k6:0.50.0"
			},
		},
		{
			name: "runner env vars",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Runner.Env = []corev1.EnvVar{
					{Name: "MY_VAR", Value: "my-value"},
					{Name: "CONN_STR", Value: "host=db;port=5432;user=k6&timeout=30s"},
				}
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
					{Name: "MY_VAR", Value: "my-value"},
					{Name: "CONN_STR", Value: "host=db;port=5432;user=k6&timeout=30s"},
				}
			},
		},
		{
			name: "parallelism with segmentation",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Parallelism = 3
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].Command = []string{
					"k6", "run", "--quiet",
					"--execution-segment=0:1/3",
					"--execution-segment-sequence=0,1/3,2/3,1",
					"/test/test.js", "--address=0.0.0.0:6565", "--paused",
					"--tag", "instance_id=1", "--tag", "testrun_name=test",
				}
			},
		},
		{
			name: "separate triggers anti-affinity",
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Separate = true
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Affinity = newAntiAffinity()
			},
		},
		{
			name:      "PLZ test run",
			tokenInfo: cloud.NewTokenInfo("plz-token-secret", "test"),
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.TestRunID = "plz-run-123"
				k6.Spec.Runner.EnvFrom = envFromConfigMap("env")
				k6.Spec.Runner.ImagePullPolicy = corev1.PullNever
				k6.Status.Conditions = []metav1.Condition{
					{
						Type:               v1alpha1.CloudPLZTestRun,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Now(),
					},
				}
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullNever
				j.Spec.Template.Spec.Containers[0].EnvFrom = envFromConfigMap("env")
				j.Spec.Template.Spec.Containers[0].Command = []string{
					"k6", "run", "--quiet", "/test/test.js", "--address=0.0.0.0:6565", "--paused",
					"--tag", "instance_id=1", "--tag", "testrun_name=test",
					"--no-setup", "--no-teardown", "--linger",
					"-e", "K6_CLOUDRUN_INSTANCE_ID=1",
				}
				j.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
					{
						Name: "K6_CLOUD_TOKEN",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "plz-token-secret"},
								Key:                  "token",
							},
						},
					},
					{Name: "K6_CLOUD_PUSH_REF_ID", Value: "plz-run-123"},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := tt.script
			if script == nil {
				script = defaultScript()
			}

			k6 := defaultTestRun()
			if tt.setupTestRun != nil {
				tt.setupTestRun(k6)
			}

			expected := defaultExpectedJob(script)
			if tt.setupExpectedJob != nil {
				tt.setupExpectedJob(expected)
			}

			tokenInfo := tt.tokenInfo
			if tokenInfo == nil {
				tokenInfo = cloud.NewTokenInfo("", "")
			}

			job, err := NewRunnerJob(k6, 1, tokenInfo)
			if err != nil {
				t.Fatalf("NewRunnerJob errored: %v", err)
			}
			if diff := deep.Equal(job, expected); diff != nil {
				t.Errorf("NewRunnerJob diff: %s", diff)
			}
		})
	}
}

func Test_NewAntiAffinity(t *testing.T) {
	expected := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: "In",
								Values:   []string{"k6"},
							},
							{
								Key:      "runner",
								Operator: "In",
								Values:   []string{"true"},
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}

	if diff := deep.Equal(newAntiAffinity(), expected); diff != nil {
		t.Errorf("newAntiAffinity() diff: %s", diff)
	}
}

func Test_NewRunnerService(t *testing.T) {
	serviceLabels := defaultLabels()
	serviceLabels["label1"] = "awesome"

	expected := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-service-1",
			Namespace:   "test",
			Labels:      serviceLabels,
			Annotations: customAnnotations(),
		},
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

	k6 := defaultTestRun()

	service, err := NewRunnerService(k6, 1)
	if err != nil {
		t.Fatalf("NewRunnerService errored: %v", err)
	}
	if diff := deep.Equal(service, expected); diff != nil {
		t.Errorf("NewRunnerService diff: %s", diff)
	}
}
