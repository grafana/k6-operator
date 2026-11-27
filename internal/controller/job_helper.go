package controllers

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// createIfNotExists creates obj unless an object of the same kind, name and
// namespace already exists. It returns true only when it created the object.
func createIfNotExists(ctx context.Context, k8sClient client.Client, obj client.Object) (bool, error) {
	existing := obj.DeepCopyObject().(client.Object)
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), existing)
	if err == nil {
		return false, nil
	}
	if !apierrors.IsNotFound(err) {
		return false, err
	}

	err = k8sClient.Create(ctx, obj)
	if apierrors.IsAlreadyExists(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func createJobIfNotExists(ctx context.Context, k8sClient client.Client, job *batchv1.Job) (bool, error) {
	return createIfNotExists(ctx, k8sClient, job)
}
