package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateJobSpecsExistingRunnerJob(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	newTestRun := func() *v1alpha1.TestRun {
		tr := &v1alpha1.TestRun{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", UID: "test-uid"},
			Spec: v1alpha1.TestRunSpec{
				Parallelism: 1,
				Script:      v1alpha1.K6Script{ConfigMap: v1alpha1.K6Configmap{Name: "test", File: "test.js"}},
			},
		}
		// A condition settled a minute ago: the guard reports a foreign job right away.
		v1alpha1.UpdateCondition(tr, v1alpha1.CloudTestRun, metav1.ConditionFalse)
		for i := range tr.Status.Conditions {
			tr.Status.Conditions[i].LastTransitionTime = metav1.NewTime(time.Now().Add(-time.Minute))
		}
		return tr
	}
	runnerJob := func() *batchv1.Job {
		return &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "test-1", Namespace: "default"}}
	}
	serviceKey := types.NamespacedName{Name: "test-service-1", Namespace: "default"}
	sti := cloud.NewSecretTokenInfo("", "default")

	t.Run("resumes when the job belongs to this test run", func(t *testing.T) {
		tr, job := newTestRun(), runnerJob()
		require.NoError(t, ctrl.SetControllerReference(tr, job, scheme))
		r := &TestRunReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(job).Build(), Scheme: scheme}

		_, recheck, err := createJobSpecs(context.Background(), logr.Discard(), tr, r, sti)

		require.NoError(t, err)
		require.False(t, recheck)
		require.NoError(t, r.Get(context.Background(), serviceKey, &corev1.Service{}))
	})

	t.Run("rejects a job that belongs to something else", func(t *testing.T) {
		tr := newTestRun()
		r := &TestRunReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(runnerJob()).Build(), Scheme: scheme}

		_, _, err := createJobSpecs(context.Background(), logr.Discard(), tr, r, sti)

		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(r.Get(context.Background(), serviceKey, &corev1.Service{})))
	})
}
