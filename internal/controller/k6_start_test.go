package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestPodReadySince(t *testing.T) {
	t.Parallel()

	since := time.Now().Add(-time.Minute).Truncate(time.Second)
	tests := []struct {
		name       string
		conditions []corev1.PodCondition
		want       bool
		wantSince  time.Time
	}{
		{name: "no conditions"},
		{name: "not ready", conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}}},
		{name: "only scheduled", conditions: []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionTrue}}},
		{
			name:       "ready",
			conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(since)}},
			want:       true,
			wantSince:  since,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pod := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: tt.conditions}}
			gotSince, got := podReadySince(pod)
			if got != tt.want {
				t.Errorf("podReadySince() ready = %v, want %v", got, tt.want)
			}
			if tt.want && !gotSince.Equal(tt.wantSince) {
				t.Errorf("podReadySince() since = %v, want %v", gotSince, tt.wantSince)
			}
		})
	}
}

// These tests run serially because stubRunnerHTTP replaces the default transport.
func TestStartJobsWaitsForReadyRunnerPods(t *testing.T) {
	for _, tt := range []struct {
		name      string
		ready     corev1.ConditionStatus
		readyAgo  []time.Duration // per pod
		wantErr   bool
		wantProbe bool
	}{
		// Services don't route to unready pods: probing them can only fail.
		{name: "running but not ready", ready: corev1.ConditionFalse, readyAgo: []time.Duration{time.Minute, time.Minute}},
		// kube-proxy may not have picked up the endpoints yet: keep waiting.
		{name: "just ready, service unreachable", ready: corev1.ConditionTrue,
			readyAgo: []time.Duration{5 * time.Second, 5 * time.Second}, wantProbe: true},
		// The deadline counts from the pod that became ready last.
		{name: "one pod just ready, service unreachable", ready: corev1.ConditionTrue,
			readyAgo: []time.Duration{6 * time.Minute, 5 * time.Second}, wantProbe: true},
		{name: "ready past deadline, service unreachable", ready: corev1.ConditionTrue,
			readyAgo: []time.Duration{6 * time.Minute, 6 * time.Minute}, wantErr: true, wantProbe: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			probes := 0
			stubRunnerHTTP(t, func(w http.ResponseWriter, r *http.Request) {
				probes++
				w.WriteHeader(http.StatusServiceUnavailable)
			})

			r, tr := runnerTestReconciler(t, 2)
			tr.Status.Stage = "created"
			v1alpha1.UpdateCondition(tr, v1alpha1.CloudPLZTestRun, metav1.ConditionFalse)
			v1alpha1.UpdateCondition(tr, v1alpha1.TestRunRunning, metav1.ConditionUnknown)
			addRunnerPods(t, r.Client, tr, tt.ready, tt.readyAgo)

			res, err := StartJobs(context.Background(), logr.Discard(), tr, r, nil)
			if tt.wantErr {
				if !errors.Is(err, errServiceNotReady) {
					t.Fatalf("err = %v, want %v", err, errServiceNotReady)
				}
			} else {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				if res.RequeueAfter != time.Second {
					t.Errorf("RequeueAfter = %v, want %v", res.RequeueAfter, time.Second)
				}
			}
			if (probes > 0) != tt.wantProbe {
				t.Errorf("service probes = %d, wantProbe %v", probes, tt.wantProbe)
			}
			if tr.Status.Stage != "created" {
				t.Errorf("stage = %q, want created", tr.Status.Stage)
			}
		})
	}
}

func addRunnerPods(t *testing.T, c client.Client, tr *v1alpha1.TestRun, ready corev1.ConditionStatus, readyAgo []time.Duration) {
	t.Helper()
	for i := 0; i < int(tr.Spec.Parallelism); i++ {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("runner-%d", i), Namespace: tr.Namespace,
				Labels: map[string]string{"app": "k6", "runner": "true", "k6_cr": tr.Name}},
		}
		if err := c.Create(context.Background(), pod); err != nil {
			t.Fatal(err)
		}
		pod.Status = corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: ready,
				LastTransitionTime: metav1.NewTime(time.Now().Add(-readyAgo[i]))}},
		}
		if err := c.Status().Update(context.Background(), pod); err != nil {
			t.Fatal(err)
		}
	}
}

// Runs serially because stubRunnerHTTP replaces the default transport.
// Takes several seconds because ResumeRunners retries with a backoff.
func TestStartJobsBuiltinStarterFailsTestRunWhenResumeFails(t *testing.T) {
	ctx := context.Background()

	var patches atomic.Int32
	stubRunnerHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patches.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	r, tr := runnerTestReconciler(t, 2)
	r.BuiltinStarter = true
	tr.Status.Stage = "created"
	v1alpha1.UpdateCondition(tr, v1alpha1.CloudPLZTestRun, metav1.ConditionFalse)
	v1alpha1.UpdateCondition(tr, v1alpha1.TestRunRunning, metav1.ConditionUnknown)
	addRunnerPods(t, r.Client, tr, corev1.ConditionTrue, []time.Duration{time.Minute, time.Minute})

	res, err := StartJobs(ctx, logr.Discard(), tr, r, nil)
	if err != nil {
		t.Fatalf("err = %v, want nil: a retry would wait for runner pods that may have finished", err)
	}
	if res != (ctrl.Result{}) {
		t.Errorf("result = %+v, want no requeue", res)
	}
	if patches.Load() == 0 {
		t.Error("runners were not resumed")
	}

	got := &v1alpha1.TestRun{}
	if err := r.Get(context.Background(), tr.NamespacedName(), got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Stage != "error" {
		t.Errorf("stage = %q, want error", got.Status.Stage)
	}
}

// Runs serially because stubRunnerHTTP replaces the default transport.
func TestStartJobsBuiltinStarterRetriesWhenResumeIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var patches atomic.Int32
	stubRunnerHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patches.Add(1)
			// as if the operator were shutting down mid-reconcile
			cancel()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	r, tr := runnerTestReconciler(t, 2)
	r.BuiltinStarter = true
	tr.Status.Stage = "created"
	v1alpha1.UpdateCondition(tr, v1alpha1.CloudPLZTestRun, metav1.ConditionFalse)
	v1alpha1.UpdateCondition(tr, v1alpha1.TestRunRunning, metav1.ConditionUnknown)
	addRunnerPods(t, r.Client, tr, corev1.ConditionTrue, []time.Duration{time.Minute, time.Minute})

	if _, err := StartJobs(ctx, logr.Discard(), tr, r, nil); err == nil {
		t.Fatal("err = nil, want an error so that the reconcile is retried")
	}
	if patches.Load() == 0 {
		t.Error("runners were not resumed")
	}

	if tr.Status.Stage != "created" {
		t.Errorf("stage = %q, want created", tr.Status.Stage)
	}
	got := &v1alpha1.TestRun{}
	if err := r.Get(context.Background(), tr.NamespacedName(), got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Stage == "error" {
		t.Error("stored stage = error, want it unchanged")
	}
}
