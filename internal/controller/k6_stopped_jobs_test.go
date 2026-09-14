package controllers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type runnerRoundTripper func(*http.Request) (*http.Response, error)

func (f runnerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// These tests run serially because the existing k6 helpers use the default transport.
func stubRunnerHTTP(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	old := http.DefaultTransport
	http.DefaultTransport = runnerRoundTripper(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		handler(w, r)
		return w.Result(), nil
	})
	t.Cleanup(func() { http.DefaultTransport = old })
}

func runnerTestReconciler(t *testing.T, count int) (*TestRunReconciler, *v1alpha1.TestRun) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	tr := &v1alpha1.TestRun{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}}
	tr.Spec.Parallelism = int32(count)
	tr.Spec.Runner.Env = []corev1.EnvVar{{Name: "K6_CLOUD_TOKEN", Value: "run-token"}}
	tr.Status.Stage, tr.Status.TestRunID = "started", "42"
	for _, condition := range []string{v1alpha1.CloudPLZTestRun, v1alpha1.TestRunRunning} {
		v1alpha1.UpdateCondition(tr, condition, metav1.ConditionTrue)
	}
	v1alpha1.UpdateCondition(tr, v1alpha1.TeardownExecuted, metav1.ConditionFalse)
	v1alpha1.UpdateCondition(tr, v1alpha1.CloudTestRunAborted, metav1.ConditionFalse)
	for i := range tr.Status.Conditions {
		tr.Status.Conditions[i].LastTransitionTime = metav1.NewTime(time.Now().Add(-time.Minute))
	}
	b := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(tr).WithObjects(tr)
	for i := 0; i < count; i++ {
		b.WithObjects(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("runner-%d", i), Namespace: tr.Namespace,
				Labels: map[string]string{"app": "k6", "runner": "true", "k6_cr": tr.Name}},
			Spec: corev1.ServiceSpec{ClusterIP: fmt.Sprintf("10.0.0.%d", i+1)},
		})
	}
	return &TestRunReconciler{Client: b.Build()}, tr
}

func TestStoppedJobsExecutionResult(t *testing.T) {
	for _, tt := range []struct {
		name, attributes string
		aborted, stopped bool
		code             cloud.ErrorCode
	}{
		{name: "old runner stopped", attributes: `"running":false`, stopped: true},
		{name: "old runner running", attributes: `"running":true`},
		{name: "pending after running becomes false", attributes: `"running":false,"execution_result":null`},
		{name: "success", attributes: `"running":false,"execution_result":{"exit_code":0}`, stopped: true},
		{name: "exception", attributes: `"running":false,"execution_result":{"exit_code":107}`, code: cloud.ScriptException},
		{name: "abort", attributes: `"running":false,"execution_result":{"exit_code":108}`, code: cloud.ScriptAborted},
		{name: "unknown result must not succeed", attributes: `"running":false,"execution_result":{"exit_code":110}`},
		{name: "malformed result", attributes: `"running":false,"execution_result":{"exit_code":"bad"}`},
		{name: "missing exit code", attributes: `"running":false,"execution_result":{}`},
		{name: "abort cleanup ignores pending result", attributes: `"running":false,"execution_result":null`, aborted: true, stopped: true},
		{name: "abort cleanup ignores error result", attributes: `"running":false,"execution_result":{"exit_code":108}`, aborted: true, stopped: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			stubRunnerHTTP(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				_, _ = fmt.Fprintf(w, `{"data":{"attributes":{%s}}}`, tt.attributes)
			})
			r, tr := runnerTestReconciler(t, 1)
			if tt.aborted {
				tr.Status.Stage = "stopped"
				v1alpha1.UpdateCondition(tr, v1alpha1.CloudTestRunAborted, metav1.ConditionTrue)
			}
			stopped, failure := StoppedJobs(context.Background(), logr.Discard(), tr, r)
			if calls != 1 || stopped != tt.stopped {
				t.Fatalf("calls=%d stopped=%t", calls, stopped)
			}
			if tt.code == 0 {
				if failure != nil {
					t.Fatalf("unexpected failure: %+v", failure)
				}
			} else if failure == nil || failure.Code != tt.code {
				t.Fatalf("failure=%+v, want code %d", failure, tt.code)
			}
		})
	}
}

func TestExecutionResultNotifiesWhileAnotherRunnerIsRunning(t *testing.T) {
	statusCalls, notifications := 0, 0
	stubRunnerHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/status":
			statusCalls++
			if r.URL.Hostname() == "10.0.0.1" {
				_, _ = fmt.Fprint(w, `{"data":{"attributes":{"running":true,"execution_result":null}}}`)
			} else {
				_, _ = fmt.Fprint(w, `{"data":{"attributes":{"running":false,"execution_result":{"exit_code":108}}}}`)
			}
		case strings.HasSuffix(r.URL.Path, "/notify"):
			notifications++
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"code":8036`) {
				t.Errorf("unexpected notification: %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = fmt.Fprint(w, `{"id":42,"run_status":2}`)
		}
	})
	r, tr := runnerTestReconciler(t, 2)
	res, err := r.reconcile(context.Background(), ctrl.Request{}, logr.Discard(), tr)
	if err != nil {
		t.Fatal(err)
	}
	if statusCalls != 2 || notifications != 1 || res.RequeueAfter == 0 || tr.Status.Stage != "started" {
		t.Fatalf("status calls=%d notifications=%d result=%+v stage=%s", statusCalls, notifications, res, tr.Status.Stage)
	}
}

func TestRunTeardownErrorClassification(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
		code   cloud.ErrorCode
	}{
		{name: "success", status: 204, code: cloud.K6OperatorStopError},
		{name: "script failure", status: 500, body: `{"errors":[{"title":"Error executing teardown","detail":"exception"}]}`, code: cloud.TeardownError},
		{name: "unavailable API", status: 503, body: `{"errors":[{"title":"Unavailable"}]}`, code: cloud.K6OperatorStopError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stubRunnerHTTP(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/teardown" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, tt.body)
			})
			code, err := runTeardown(context.Background(), []string{"runner"}, logr.Discard())
			if (err != nil) != (tt.status >= 400) || code != tt.code {
				t.Fatalf("error=%v code=%d", err, code)
			}
		})
	}
	if code, err := runTeardown(context.Background(), nil, logr.Discard()); err == nil || code != cloud.K6OperatorStopError {
		t.Fatalf("no runner: error=%v code=%d", err, code)
	}
	old := http.DefaultTransport
	http.DefaultTransport = runnerRoundTripper(func(*http.Request) (*http.Response, error) { return nil, errors.New("connection lost") })
	t.Cleanup(func() { http.DefaultTransport = old })
	if code, err := runTeardown(context.Background(), []string{"runner"}, logr.Discard()); err == nil || code != cloud.K6OperatorStopError {
		t.Fatalf("network failure: error=%v code=%d", err, code)
	}
}
