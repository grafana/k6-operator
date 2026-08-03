package plz

import (
	"fmt"
	rand "math/rand/v2"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/resources/containers"
	"github.com/grafana/k6-operator/pkg/testrun"
	"go.k6.io/k6/v2/cloudapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// it should be safe to call StartFactory more than once
func Test_StartFactory_idempotent(t *testing.T) {
	c, _ := client.New(nil, client.Options{})
	worker := NewPLZWorker(&v1alpha1.PrivateLoadZone{}, "token", c, logr.Logger{})

	worker.StartFactory()
	ptrPoller := worker.poller
	nGoroutines := runtime.NumGoroutine()

	time.Sleep(time.Duration(rand.IntN(2000)) * time.Millisecond)
	worker.StartFactory()

	if worker.poller != ptrPoller {
		t.Errorf("address of the poller changed during idempotent call")
	}
	if nGoroutines != runtime.NumGoroutine() {
		t.Errorf("number of goroutines changed during idempotent call")
	}
}

// it should be safe to call StopFactory more than once
func Test_StopFactory_idempotent(t *testing.T) {
	c, _ := client.New(nil, client.Options{})
	worker := NewPLZWorker(&v1alpha1.PrivateLoadZone{}, "token", c, logr.Logger{})

	worker.StartFactory()
	worker.StopFactory()

	if worker.poller.IsPolling() {
		t.Errorf("poller shouldn't be polling after the 1st StopFactory")
	}

	time.Sleep(time.Duration(rand.IntN(2000)) * time.Millisecond)
	worker.StopFactory()

	if worker.poller.IsPolling() {
		t.Errorf("poller shouldn't be polling after the 2nd StopFactory")
	}
}

func Test_plzk6Args(t *testing.T) {
	tests := []struct {
		name      string
		plzName   string
		testRunID string
		trData    *cloud.TestRunData
		expected  []string
	}{
		{
			name:      "default, hard-coded values",
			plzName:   "",
			testRunID: "0",
			trData:    &cloud.TestRunData{},
			expected: []string{
				"--out",
				"cloud",
				"--no-thresholds",
				"--log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=,label.test_run_id=0,header.Authorization=Token $(K6_CLOUD_TOKEN)",
				"--include-system-env-vars=false",
				"--verbose",
				"--summary-mode=disabled",
			},
		},
		{
			name:      "arguments from preprocessing",
			plzName:   "private-zone",
			testRunID: "1234",
			trData: &cloud.TestRunData{
				TagArgs: []string{"--tag", "load_zone=private-zone", "--tag", "team=operator"},
				EnvArgs: []string{"-e", "K6_CLOUDRUN_TEST_RUN_ID=1234"},
				LZConfig: cloud.LZConfig{
					CLIArgs: cloud.CLIArgs{IncludeSystemEnvVars: true},
				},
			},
			expected: []string{
				"--out",
				"cloud",
				"--tag",
				"load_zone=private-zone",
				"--tag",
				"team=operator",
				"--no-thresholds",
				"--log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=private-zone,label.test_run_id=1234,header.Authorization=Token $(K6_CLOUD_TOKEN)",
				"-e",
				"K6_CLOUDRUN_TEST_RUN_ID=1234",
				"--include-system-env-vars=true",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if diff := deep.Equal(test.expected, plzk6Args(test.plzName, test.testRunID, test.trData)); diff != nil {
				t.Error(diff)
			}
		})
	}
}

func Test_complete_correctDefinitionOfTestRun(t *testing.T) {
	// The following are the definitions that
	// are expected from PLZ worker now.

	var (
		mainIngest = "https://ingest.k6.io"

		volumeMount = corev1.VolumeMount{
			Name:      "archive-volume",
			MountPath: "/test",
		}
		// zero-values test run definition
		defaultTestRun = v1alpha1.TestRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: testrun.PLZTestName("0"),
			},
			Spec: v1alpha1.TestRunSpec{
				Runner: v1alpha1.Pod{
					Volumes: []corev1.Volume{{
						Name: "archive-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					},
					VolumeMounts: []corev1.VolumeMount{volumeMount},
					InitContainers: []v1alpha1.InitContainer{
						containers.NewS3InitContainer(
							"",
							"ghcr.io/grafana/k6-operator:latest-starter",
							volumeMount,
						),
					},
					EnvFrom: []corev1.EnvFromSource{},
				},
				Script: v1alpha1.K6Script{
					LocalFile: "/test/archive.tar",
				},
				Parallelism: int32(0),
				Separate:    false,
				Args:        plzk6Args("", "0", &cloud.TestRunData{}),
				Cleanup:     v1alpha1.Cleanup("post"),

				TestRunID: "0",
			},
		}

		// non-empty values to use in test cases
		someToken        = "some-token"
		someSA           = "some-service-account"
		someNodeSelector = map[string]string{"foo": "bar"}
		someNS           = "some-ns"
		resourceLimits   = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("1G"),
		}
		somePLZName     = "my-plz"
		someTestRunID   = 6543
		someRunnerImage = "grafana/k6:0.52.0"
		someInstances   = 10
		someArchiveURL  = "https://foo.s3.amazonaws.com"
		someEnvVars     = map[string]string{
			"ENV": "hello world",
			"foo": "bar",
		}
		someLZDistribution = cloud.LZDistribution{
			"some-label": cloud.Distribution{LoadZone: "some-zone", Percent: 100},
		}
		// podTemplate test values
		someAllowPrivEscalation       = false
		someRunAsUser           int64 = 1000
		someTolerations               = []corev1.Toleration{
			{
				Key:      "dedicated",
				Operator: corev1.TolerationOpEqual,
				Value:    "k6",
				Effect:   corev1.TaintEffectNoSchedule,
			},
		}
		someContainerSecCtx = corev1.SecurityContext{
			AllowPrivilegeEscalation: &someAllowPrivEscalation,
		}
		somePodSecCtx = corev1.PodSecurityContext{
			RunAsUser: &someRunAsUser,
		}

		// TestRuns expected in different cases;
		// see how they are populated below

		requiredFieldsTestRun = defaultTestRun
		optionalFieldsTestRun = defaultTestRun //nolint:ineffassign
		cloudFieldsTestRun    = defaultTestRun //nolint:ineffassign
		cloudEnvVarsTestRun   = defaultTestRun //nolint:ineffassign

		// TestRuns expected for podTemplate cases
		podTemplateTolerationsTestRun     = defaultTestRun //nolint:ineffassign
		podTemplateContainerSecCtxTestRun = defaultTestRun //nolint:ineffassign
		podTemplatePodSecCtxTestRun       = defaultTestRun //nolint:ineffassign
		podTemplateAllTestRun             = defaultTestRun //nolint:ineffassign

		// TestRuns expected for secrets cases
		secretsWithTokenTestRun = defaultTestRun //nolint:ineffassign

		// TestRun expected when include_system_env_vars is true
		includeSysEnvVarsTestRun = defaultTestRun //nolint:ineffassign
	)

	someLZTagArgs := []string{"--tag", "load_zone=some-zone"}
	cloudExecEnvArgs := []string{
		"-e", "K6_CLOUDRUN_DISTRIBUTION=some-label",
		"-e", "K6_CLOUDRUN_LOAD_ZONE=some-zone",
		"-e", "K6_CLOUDRUN_TEST_RUN_ID=6543",
	}
	cloudEnvVarsEnvArgs := []string{
		"-e", "ENV=hello world",
		"-e", "K6_CLOUDRUN_DISTRIBUTION=some-label",
		"-e", "K6_CLOUDRUN_LOAD_ZONE=some-zone",
		"-e", "K6_CLOUDRUN_TEST_RUN_ID=6543",
		"-e", "foo=bar",
	}

	// populate TestRuns for different test cases

	requiredFieldsTestRun.Spec.Token = someToken
	requiredFieldsTestRun.Spec.Runner.Resources.Limits = resourceLimits

	optionalFieldsTestRun = requiredFieldsTestRun // build up on top of required field case
	optionalFieldsTestRun.Namespace = someNS
	optionalFieldsTestRun.Spec.Runner.ServiceAccountName = someSA
	optionalFieldsTestRun.Spec.Runner.NodeSelector = someNodeSelector
	optionalFieldsTestRun.Spec.Starter.ServiceAccountName = someSA
	optionalFieldsTestRun.Spec.Starter.NodeSelector = someNodeSelector

	cloudFieldsTestRun = requiredFieldsTestRun // build up on top of required field case
	cloudFieldsTestRun.Name = testrun.PLZTestName(fmt.Sprintf("%d", someTestRunID))
	cloudFieldsTestRun.Spec.TestRunID = fmt.Sprintf("%d", someTestRunID)
	cloudFieldsTestRun.Spec.Args = plzk6Args(
		"",
		fmt.Sprintf("%d", someTestRunID),
		&cloud.TestRunData{
			TagArgs: someLZTagArgs,
			EnvArgs: cloudExecEnvArgs,
		})
	cloudFieldsTestRun.Spec.Runner.InitContainers = []v1alpha1.InitContainer{
		containers.NewS3InitContainer(
			someArchiveURL,
			"ghcr.io/grafana/k6-operator:latest-starter",
			volumeMount,
		),
	}
	cloudFieldsTestRun.Spec.Runner.Image = someRunnerImage
	cloudFieldsTestRun.Spec.Parallelism = int32(someInstances)
	cloudFieldsTestRun.Spec.Runner.Env = append([]corev1.EnvVar{}, cloud.AggregationEnvVars(&cloudapi.Config{})...)
	cloudFieldsTestRun.Spec.Runner.Env = append(
		cloudFieldsTestRun.Spec.Runner.Env,
		corev1.EnvVar{Name: "K6_CLOUD_HOST", Value: mainIngest},
	)

	cloudEnvVarsTestRun = cloudFieldsTestRun // build up on top of cloud fields case
	cloudEnvVarsTestRun.Spec.Args = plzk6Args(
		somePLZName,
		fmt.Sprintf("%d", someTestRunID),
		&cloud.TestRunData{
			TagArgs: someLZTagArgs,
			EnvArgs: cloudEnvVarsEnvArgs,
		})
	cloudEnvVarsTestRun.Spec.Runner.Env = []corev1.EnvVar{
		{Name: "K6_USER_AGENT", Value: "Grafana Cloud k6"},
	}
	cloudEnvVarsTestRun.Spec.Runner.Env = append(
		cloudEnvVarsTestRun.Spec.Runner.Env,
		cloud.AggregationEnvVars(&cloudapi.Config{})...,
	)
	cloudEnvVarsTestRun.Spec.Runner.Env = append(
		cloudEnvVarsTestRun.Spec.Runner.Env,
		corev1.EnvVar{Name: "K6_CLOUD_HOST", Value: mainIngest},
	)

	podTemplateTolerationsTestRun = requiredFieldsTestRun
	podTemplateTolerationsTestRun.Spec.Runner.Tolerations = someTolerations
	podTemplateTolerationsTestRun.Spec.Starter.Tolerations = someTolerations

	podTemplateContainerSecCtxTestRun = requiredFieldsTestRun
	podTemplateContainerSecCtxTestRun.Spec.Runner.ContainerSecurityContext = someContainerSecCtx
	podTemplateContainerSecCtxTestRun.Spec.Starter.ContainerSecurityContext = someContainerSecCtx

	podTemplatePodSecCtxTestRun = requiredFieldsTestRun
	podTemplatePodSecCtxTestRun.Spec.Runner.SecurityContext = somePodSecCtx
	podTemplatePodSecCtxTestRun.Spec.Starter.SecurityContext = somePodSecCtx

	podTemplateAllTestRun = requiredFieldsTestRun
	podTemplateAllTestRun.Spec.Runner.Tolerations = someTolerations
	podTemplateAllTestRun.Spec.Starter.Tolerations = someTolerations
	podTemplateAllTestRun.Spec.Runner.ContainerSecurityContext = someContainerSecCtx
	podTemplateAllTestRun.Spec.Starter.ContainerSecurityContext = someContainerSecCtx
	podTemplateAllTestRun.Spec.Runner.SecurityContext = somePodSecCtx
	podTemplateAllTestRun.Spec.Starter.SecurityContext = somePodSecCtx

	includeSysEnvVarsTestRun = cloudFieldsTestRun
	includeSysEnvVarsTestRun.Spec.Args = plzk6Args(
		"",
		fmt.Sprintf("%d", someTestRunID),
		&cloud.TestRunData{
			TagArgs:  someLZTagArgs,
			EnvArgs:  cloudExecEnvArgs,
			LZConfig: cloud.LZConfig{CLIArgs: cloud.CLIArgs{IncludeSystemEnvVars: true}},
		})

	someSecretsConfig := &cloud.SecretsConfig{
		Endpoint:     "https://api.k6.io/provisioning/v1/test_runs/6543/decrypt_secret?name={key}",
		ResponsePath: "plaintext",
	}
	someTestRunToken := "abc123token"

	secretsWithTokenTestRun = cloudFieldsTestRun
	secretsWithTokenTestRun.Spec.Runner.Env = append([]corev1.EnvVar{}, cloud.AggregationEnvVars(&cloudapi.Config{})...)
	secretsWithTokenTestRun.Spec.Runner.Env = append(
		secretsWithTokenTestRun.Spec.Runner.Env,
		corev1.EnvVar{Name: "K6_SECRET_SOURCE", Value: "url"},
		corev1.EnvVar{Name: "K6_SECRET_SOURCE_URL_URL_TEMPLATE", Value: someSecretsConfig.Endpoint},
		corev1.EnvVar{Name: "K6_SECRET_SOURCE_URL_RESPONSE_PATH", Value: someSecretsConfig.ResponsePath},
		corev1.EnvVar{Name: "K6_SECRET_SOURCE_URL_HEADER_AUTHORIZATION", Value: "Bearer " + someTestRunToken},
		corev1.EnvVar{Name: "K6_CLOUD_HOST", Value: mainIngest},
	)

	testCases := []struct {
		name      string
		plz       *v1alpha1.PrivateLoadZone
		cloudData *cloud.TestRunData
		ingestUrl string
		expected  *v1alpha1.TestRun
	}{
		{
			name:      "empty input gets a zero-values TestRun",
			plz:       &v1alpha1.PrivateLoadZone{},
			cloudData: &cloud.TestRunData{},
			ingestUrl: mainIngest,
			expected:  &defaultTestRun,
		},
		{
			name: "required fields in PLZ",
			plz: &v1alpha1.PrivateLoadZone{
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
				},
			},
			cloudData: &cloud.TestRunData{},
			ingestUrl: mainIngest,
			expected:  &requiredFieldsTestRun,
		},
		{
			name: "optional fields in PLZ",
			plz: &v1alpha1.PrivateLoadZone{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: someNS,
				},
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
					ServiceAccountName: someSA,
					NodeSelector:       someNodeSelector,
				},
			},
			cloudData: &cloud.TestRunData{},
			ingestUrl: mainIngest,
			expected:  &optionalFieldsTestRun,
		},
		{
			name: "basic cloud fields",
			plz: &v1alpha1.PrivateLoadZone{
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
				},
			},
			cloudData: &cloud.TestRunData{
				TestRunId: someTestRunID,
				LZConfig: cloud.LZConfig{
					RunnerImage:   someRunnerImage,
					InstanceCount: someInstances,
					ArchiveURL:    someArchiveURL,
				},
				LZDistribution: someLZDistribution,
			},
			ingestUrl: mainIngest,
			expected:  &cloudFieldsTestRun,
		},
		{
			name: "cloud fields with env vars",
			plz: &v1alpha1.PrivateLoadZone{
				ObjectMeta: metav1.ObjectMeta{Name: somePLZName},
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
				},
			},
			cloudData: &cloud.TestRunData{
				TestRunId: someTestRunID,
				LZConfig: cloud.LZConfig{
					RunnerImage:   someRunnerImage,
					InstanceCount: someInstances,
					ArchiveURL:    someArchiveURL,
					Environment:   someEnvVars,
					CLIArgs: cloud.CLIArgs{
						UserAgent: "Grafana Cloud k6",
					},
				},
				LZDistribution: someLZDistribution,
			},
			ingestUrl: mainIngest,
			expected:  &cloudEnvVarsTestRun,
		},
		{
			name: "podTemplate with tolerations",
			plz: &v1alpha1.PrivateLoadZone{
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
					PodTemplate: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Tolerations: someTolerations,
						},
					},
				},
			},
			cloudData: &cloud.TestRunData{},
			ingestUrl: mainIngest,
			expected:  &podTemplateTolerationsTestRun,
		},
		{
			name: "podTemplate with container security context",
			plz: &v1alpha1.PrivateLoadZone{
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
					PodTemplate: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "k6",
									SecurityContext: &someContainerSecCtx,
								},
							},
						},
					},
				},
			},
			cloudData: &cloud.TestRunData{},
			ingestUrl: mainIngest,
			expected:  &podTemplateContainerSecCtxTestRun,
		},
		{
			name: "podTemplate with pod security context",
			plz: &v1alpha1.PrivateLoadZone{
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
					PodTemplate: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							SecurityContext: &somePodSecCtx,
						},
					},
				},
			},
			cloudData: &cloud.TestRunData{},
			ingestUrl: mainIngest,
			expected:  &podTemplatePodSecCtxTestRun,
		},
		{
			name: "podTemplate with tolerations, container and pod security context",
			plz: &v1alpha1.PrivateLoadZone{
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
					PodTemplate: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Tolerations:     someTolerations,
							SecurityContext: &somePodSecCtx,
							Containers: []corev1.Container{
								{
									Name:            "k6",
									SecurityContext: &someContainerSecCtx,
								},
							},
						},
					},
				},
			},
			cloudData: &cloud.TestRunData{},
			ingestUrl: mainIngest,
			expected:  &podTemplateAllTestRun,
		},
		{
			name: "include_system_env_vars is set to true",
			plz: &v1alpha1.PrivateLoadZone{
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
				},
			},
			cloudData: &cloud.TestRunData{
				TestRunId: someTestRunID,
				LZConfig: cloud.LZConfig{
					RunnerImage:   someRunnerImage,
					InstanceCount: someInstances,
					ArchiveURL:    someArchiveURL,
					CLIArgs: cloud.CLIArgs{
						IncludeSystemEnvVars: true,
					},
				},
				LZDistribution: someLZDistribution,
			},
			ingestUrl: mainIngest,
			expected:  &includeSysEnvVarsTestRun,
		},
		{
			name: "secrets config with token",
			plz: &v1alpha1.PrivateLoadZone{
				Spec: v1alpha1.PrivateLoadZoneSpec{
					Token: someToken,
					Resources: corev1.ResourceRequirements{
						Limits: resourceLimits,
					},
				},
			},
			cloudData: &cloud.TestRunData{
				TestRunId: someTestRunID,
				LZConfig: cloud.LZConfig{
					RunnerImage:   someRunnerImage,
					InstanceCount: someInstances,
					ArchiveURL:    someArchiveURL,
				},
				SecretsToken:   someTestRunToken,
				SecretsConfig:  someSecretsConfig,
				LZDistribution: someLZDistribution,
			},
			ingestUrl: mainIngest,
			expected:  &secretsWithTokenTestRun,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			c, _ := client.New(nil, client.Options{})
			worker := NewPLZWorker(testCase.plz, "token", c, logr.Logger{})

			tr := worker.template.Create()

			// Preprocess must succeed only when a single-LZ distribution is provided.
			err := testCase.cloudData.Preprocess()
			if expectingError := (len(testCase.cloudData.LZDistribution) != 1); expectingError && (err == nil) {
				t.Errorf("Preprocess() error = %v, expecting error: %v", err, expectingError)
			}

			worker.complete(tr, testCase.cloudData)

			if diff := deep.Equal(tr, testCase.expected); diff != nil {
				t.Errorf("worker.complete returned unexpected data, diff: %s", diff)
			}
		})
	}
}

// scheme is a global var that is used for only `ctrl.SetControllerReference“ call
// by PLZworker; so it makes sense to check its safety for concurrent execution.
func Test_scheme_threadSafety(t *testing.T) {
	SetScheme(k8sruntime.NewScheme())
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			time.Sleep(time.Duration(rand.IntN(2000)) * time.Millisecond)

			var p1, p2 corev1.Pod
			_ = ctrl.SetControllerReference(&p1, &p2, scheme)
			wg.Done()
		}()
	}

	wg.Wait()
}
