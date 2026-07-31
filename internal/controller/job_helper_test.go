package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateJobIfNotExists(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, batchv1.AddToScheme(scheme))

	t.Run("creates a missing job", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-starter",
				Namespace: "default",
			},
		}

		created, err := createJobIfNotExists(context.Background(), k8sClient, job)

		require.NoError(t, err)
		require.True(t, created)
		require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(job), &batchv1.Job{}))
	})

	t.Run("keeps an existing job", func(t *testing.T) {
		existing := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-stopper",
				Namespace: "default",
			},
		}
		k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

		created, err := createJobIfNotExists(context.Background(), k8sClient, existing.DeepCopy())

		require.NoError(t, err)
		require.False(t, created)
	})
}
