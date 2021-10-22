package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile k6 status with job status
func ReconcileJobs(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler) (ctrl.Result, error) {
	selector := labels.SelectorFromSet(map[string]string{
		"app":   "k6",
		"k6_cr": k6.Name,
	})

	opts := &client.ListOptions{LabelSelector: selector, Namespace: k6.Namespace}
	jl := &batchv1.JobList{}

	if err := r.List(ctx, jl, opts); err != nil {
		log.Error(err, "Could not list jobs")
		return ctrl.Result{}, err
	}

	//TODO: We should distinguish between suceeded/failed
	var finished int32
	for _, job := range jl.Items {
		if job.Status.Active != 0 {
			continue
		}
		finished++
	}

	log.Info(fmt.Sprintf("%d/%d jobs complete", finished, k6.Spec.Parallelism+1))

	// parallelism (pods) + starter pod = total expected
	if finished == k6.Spec.Parallelism+1 {
		k6.Status.Stage = "finished"
		if err := r.Client.Status().Update(ctx, k6); err != nil {
			log.Error(err, "Could not update status of custom resource")
		}
	}

	return ctrl.Result{}, nil
}
