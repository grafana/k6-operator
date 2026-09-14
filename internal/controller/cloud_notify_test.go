package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/grafana/k6-operator/pkg/cloud"
	"go.k6.io/k6/v2/errext/exitcodes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunnerExitError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		pods            []corev1.Pod
		beforeExecution bool
		wantCode        cloud.ErrorCode
	}{
		{name: "no pods"},
		{name: "external abort", pods: []corev1.Pod{terminatedPod("runner", int32(exitcodes.ExternalAbort), "")}, wantCode: cloud.K6OperatorRunnerError},
		{name: "invalid config", pods: []corev1.Pod{terminatedPod("runner", int32(exitcodes.InvalidConfig), "")}, wantCode: cloud.K6OperatorStartError},
		{name: "no normal exit", pods: []corev1.Pod{terminatedPod("runner", -1, "")}, wantCode: cloud.K6OperatorRunnerError},
		{name: "exit code 255 remains unmapped", pods: []corev1.Pod{terminatedPod("runner", 255, "")}},
		{name: "no uint8 truncation", pods: []corev1.Pod{terminatedPod("runner", 363, "")}},
		{name: "skip unmapped runner", pods: []corev1.Pod{terminatedPod("runner-1", 137, "OOMKilled"), terminatedPod("runner-2", 108, "")}, wantCode: cloud.ScriptAborted},

		{name: "running pod", pods: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "runner-1"}}}},
		{name: "zero exit", pods: []corev1.Pod{terminatedPod("runner-1", 0, "")}},
		{
			name:            "before execution",
			pods:            []corev1.Pod{terminatedPod("runner-1", int32(exitcodes.ScriptException), "")},
			beforeExecution: true,
			wantCode:        cloud.K6OperatorStartError,
		},
		{
			name:     "script exception",
			pods:     []corev1.Pod{terminatedPod("runner-1", int32(exitcodes.ScriptException), "")},
			wantCode: cloud.ScriptException,
		},
		{
			name:     "script abort",
			pods:     []corev1.Pod{terminatedPod("runner-1", int32(exitcodes.ScriptAborted), "")},
			wantCode: cloud.ScriptAborted,
		},
		{
			name:     "panic",
			pods:     []corev1.Pod{terminatedPod("runner-1", int32(exitcodes.GoPanic), "")},
			wantCode: cloud.UnknownError,
		},
		{
			name:     "OOM deferred",
			pods:     []corev1.Pod{terminatedPod("runner-1", 137, "Error")},
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got *cloud.TestRunNotification
			for _, pod := range tt.pods {
				if got = runnerExitError(pod, tt.beforeExecution); got != nil {
					break
				}
			}
			if tt.wantCode == 0 {
				if got != nil {
					t.Fatalf("runnerExitError() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("runnerExitError() = nil")
			}
			if got.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", got.Code, tt.wantCode)
			}
			if got.Detail == "" {
				t.Error("detail is empty")
			}
		})
	}
}

func terminatedPod(name string, exitCode int32, reason string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "k6",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: exitCode,
						Reason:   reason,
					},
				},
			}},
		},
	}
}

func TestRunnerExitIgnoresSidecars(t *testing.T) {
	t.Parallel()
	pod := terminatedPod("runner", int32(exitcodes.ScriptException), "Error")
	pod.Status.ContainerStatuses[0].Name = "sidecar"
	pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{
		Name: "k6", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}},
	})
	if got := runnerExitError(pod, false); got != nil {
		t.Fatalf("reported sidecar exit: %+v", got)
	}
}

func TestCloudNotificationRouting(t *testing.T) {
	t.Parallel()
	for _, plz := range []bool{false, true} {
		name := "cloud output"
		if plz {
			name = "PLZ"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			requests := make(chan string, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- r.URL.Path
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			tr := &v1alpha1.TestRun{}
			tr.Status.TestRunID = "42"
			tr.Spec.Runner.Env = []corev1.EnvVar{{Name: "K6_CLOUD_TOKEN", Value: "run-token"}}
			v1alpha1.UpdateCondition(tr, v1alpha1.CloudTestRun, metav1.ConditionTrue)
			if plz {
				v1alpha1.UpdateCondition(tr, v1alpha1.CloudPLZTestRun, metav1.ConditionTrue)
			}
			c := cloud.NewClient(logr.Discard(), "legacy-token", server.URL)
			sendCloudError(context.Background(), logr.Discard(), tr, c, cloud.SetupError, "setup failed")
			if err := finishCloudTestRun(context.Background(), tr, c); err != nil {
				t.Fatal(err)
			}
			want := []string{"/orchestrator/v1/testruns/42/events", "/v1/tests/42"}
			if plz {
				want = []string{"/provisioning/v1/test_runs/42/notify", "/provisioning/v1/test_runs/42/notify"}
			}
			for _, path := range want {
				select {
				case got := <-requests:
					if got != path {
						t.Errorf("path = %s, want %s", got, path)
					}
				default:
					t.Fatal("missing request")
				}
			}
		})
	}
}

// if one runner failed during PLZ test run, it should be reported
// as either ScriptException or K6OperatorStartError to notify endpoint
func TestPLZTerminatedRunner(t *testing.T) {
	t.Parallel()
	for _, stage := range []v1alpha1.Stage{"created", "started"} {
		t.Run(string(stage), func(t *testing.T) {
			t.Parallel()
			codes := make(chan cloud.ErrorCode, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Error cloud.TestRunNotification `json:"error"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				codes <- body.Error.Code
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			tr := &v1alpha1.TestRun{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}}
			tr.Status.Stage, tr.Status.TestRunID = stage, "42"
			tr.Spec.Parallelism = 2
			tr.Spec.Runner.Env = []corev1.EnvVar{{Name: "K6_CLOUD_TOKEN", Value: "run-token"}, {Name: "K6_CLOUD_HOST", Value: server.URL}}
			v1alpha1.UpdateCondition(tr, v1alpha1.CloudPLZTestRun, metav1.ConditionTrue)
			failed := terminatedPod("failed", int32(exitcodes.ScriptException), "Error")
			failed.Namespace = tr.Namespace
			failed.Labels = map[string]string{"app": "k6", "k6_cr": tr.Name, "runner": "true"}
			running := failed.DeepCopy()
			running.Name = "running"
			running.Status.ContainerStatuses[0].State = corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
			scheme := runtime.NewScheme()
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			r := &TestRunReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(&failed, running).Build()}
			result, err := r.reconcile(context.Background(), ctrl.Request{}, logr.Discard(), tr)
			if err != nil {
				t.Fatal(err)
			}
			if result.RequeueAfter == 0 || tr.Status.Stage != stage {
				t.Fatal("must wait for backend abort without changing stage")
			}
			want := cloud.ScriptException
			if stage == "created" {
				want = cloud.K6OperatorStartError
			}
			select {
			case got := <-codes:
				if got != want {
					t.Errorf("code = %d, want %d", got, want)
				}
			default:
				t.Fatal("terminated runner did not notify while another runner was running")
			}
		})
	}
}
