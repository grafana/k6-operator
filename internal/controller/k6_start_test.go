package controllers

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Runs serially because stubRunnerHTTP replaces the default transport.
func TestStartJobsBuiltinStarterFailsTestRunWhenResumeFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var patches atomic.Int32
	stubRunnerHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patches.Add(1)
			// skip ResumeRunners' retry backoff
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
	for i := 0; i < int(tr.Spec.Parallelism); i++ {
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("runner-%d", i), Namespace: tr.Namespace,
			Labels: map[string]string{"app": "k6", "runner": "true", "k6_cr": tr.Name}}}
		if err := r.Create(context.Background(), pod); err != nil {
			t.Fatal(err)
		}
		pod.Status.Phase = corev1.PodRunning
		if err := r.Status().Update(context.Background(), pod); err != nil {
			t.Fatal(err)
		}
	}

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
