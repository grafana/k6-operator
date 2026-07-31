package controllers

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createJobIfNotExists(ctx context.Context, k8sClient client.Client, job *batchv1.Job) (bool, error) {
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(job), &batchv1.Job{})
	if err == nil {
		return false, nil
	}
	if !apierrors.IsNotFound(err) {
		return false, err
	}

	err = k8sClient.Create(ctx, job)
	if apierrors.IsAlreadyExists(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
