package plz

import (
	"fmt"
	rand "math/rand/v2"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/resources/containers"
	"github.com/grafana/k6-operator/pkg/testrun"
	"go.k6.io/k6/cloudapi"
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
					Env: append([]corev1.EnvVar{{
						Name:  "K6_CLOUD_HOST",
						Value: mainIngest,
					}}, cloud.AggregationEnvVars(&cloudapi.Config{})...),
					EnvFrom: []corev1.EnvFromSource{},
				},
				Script: v1alpha1.K6Script{
					LocalFile: "/test/archive.tar",
				},
				Parallelism: int32(0),
				Separate:    false,
				Arguments:   "--out cloud --no-thresholds --log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=,label.test_run_id=0,header.Authorization=\"Token $(K6_CLOUD_TOKEN)\"",
				Cleanup:     v1alpha1.Cleanup("post"),

				TestRunID: "0",
			},
		}

		// non-empty values to use int test cases
		someToken        = "some-token"
		someSA           = "some-service-account"
		someNodeSelector = map[string]string{"foo": "bar"}
		someNS           = "some-ns"
		resourceLimits   = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("1G"),
		}
		someTestRunID   = 6543
		someRunnerImage = "grafana/k6:0.52.0"
		someInstances   = 10
		someArchiveURL  = "https://foo.s3.amazonaws.com"
		someEnvVars     = map[string]string{
			"ENV": "VALUE",
			"foo": "bar",
		}

		// TestRuns expected in different cases;
		// see how they are populated below
		requiredFieldsTestRun = defaultTestRun
		optionalFieldsTestRun = defaultTestRun //nolint:ineffassign
		cloudFieldsTestRun    = defaultTestRun //nolint:ineffassign
		cloudEnvVarsTestRun   = defaultTestRun //nolint:ineffassign
	)

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
	cloudFieldsTestRun.Spec.Arguments = strings.Replace(requiredFieldsTestRun.Spec.Arguments,
		"test_run_id=0",
		fmt.Sprintf("test_run_id=%d", someTestRunID),
		1)
	cloudFieldsTestRun.Spec.Runner.InitContainers = []v1alpha1.InitContainer{
		containers.NewS3InitContainer(
			someArchiveURL,
			"ghcr.io/grafana/k6-operator:latest-starter",
			volumeMount,
		),
	}
	cloudFieldsTestRun.Spec.Runner.Image = someRunnerImage
	cloudFieldsTestRun.Spec.Parallelism = int32(someInstances)

	cloudEnvVarsTestRun = cloudFieldsTestRun // build up on top of cloud fields case
	cloudEnvVarsTestRun.Spec.Runner.Env = append([]corev1.EnvVar{
		{
			Name:  "ENV",
			Value: "VALUE",
		},
		{
			Name:  "foo",
			Value: "bar",
		},
	}, defaultTestRun.Spec.Runner.Env...)

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
			},
			ingestUrl: mainIngest,
			expected:  &cloudFieldsTestRun,
		},
		{
			name: "cloud fields with env vars",
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
					Environment:   someEnvVars,
				},
			},
			ingestUrl: mainIngest,
			expected:  &cloudEnvVarsTestRun,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			c, _ := client.New(nil, client.Options{})
			worker := NewPLZWorker(testCase.plz, "token", c, logr.Logger{})

			tr := worker.template.Create()
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
