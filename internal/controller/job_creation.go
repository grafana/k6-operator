package controllers

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createJobIfNotExists(ctx context.Context, c client.Client, job *batchv1.Job) (bool, error) {
	existing := &batchv1.Job{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(job), existing); err == nil {
		return false, nil
	} else if !apierrors.IsNotFound(err) {
		return false, err
	}

	if err := c.Create(ctx, job); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
