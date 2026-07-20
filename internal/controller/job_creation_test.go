package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateJobIfNotExistsSkipsExistingJob(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, batchv1.AddToScheme(scheme))

	existing := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "starter", Namespace: "default"}}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

	created, err := createJobIfNotExists(context.Background(), client, existing.DeepCopy())

	require.NoError(t, err)
	require.False(t, created)
}

func TestCreateJobIfNotExistsCreatesMissingJob(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, batchv1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "starter", Namespace: "default"}}

	created, err := createJobIfNotExists(context.Background(), k8sClient, job)

	require.NoError(t, err)
	require.True(t, created)
	stored := &batchv1.Job{}
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(job), stored))
}

type failingGetClient struct {
	client.Client
	err error
}

func (c failingGetClient) Get(context.Context, types.NamespacedName, client.Object, ...client.GetOption) error {
	return c.err
}

func TestCreateJobIfNotExistsReturnsLookupError(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, batchv1.AddToScheme(scheme))

	lookupErr := errors.New("lookup failed")
	baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := failingGetClient{Client: baseClient, err: lookupErr}
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "starter", Namespace: "default"}}

	created, err := createJobIfNotExists(context.Background(), k8sClient, job)

	require.ErrorIs(t, err, lookupErr)
	require.False(t, created)
}

type jobCreatedDuringLookupClient struct {
	client.Client
}

func (c jobCreatedDuringLookupClient) Get(context.Context, types.NamespacedName, client.Object, ...client.GetOption) error {
	return apierrors.NewNotFound(schema.GroupResource{Group: "batch", Resource: "jobs"}, "starter")
}

func (c jobCreatedDuringLookupClient) Create(context.Context, client.Object, ...client.CreateOption) error {
	return apierrors.NewAlreadyExists(schema.GroupResource{Group: "batch", Resource: "jobs"}, "starter")
}

func TestCreateJobIfNotExistsHandlesConcurrentCreation(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, batchv1.AddToScheme(scheme))

	baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := jobCreatedDuringLookupClient{Client: baseClient}
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "starter", Namespace: "default"}}

	created, err := createJobIfNotExists(context.Background(), k8sClient, job)

	require.NoError(t, err)
	require.False(t, created)
}
