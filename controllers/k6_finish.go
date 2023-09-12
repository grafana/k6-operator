package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FinishJobs checks if the runners pods have finished execution.
func FinishJobs(ctx context.Context, log logr.Logger, k6 v1alpha1.TestRunI, r *TestRunReconciler) (allFinished bool) {
	if len(k6.GetStatus().TestRunID) > 0 {
		log = log.WithValues("testRunId", k6.GetStatus().TestRunID)
	}

	log.Info("Checking if all runner pods are finished")

	selector := labels.SelectorFromSet(map[string]string{
		"app":    "k6",
		"k6_cr":  k6.NamespacedName().Name,
		"runner": "true",
	})

	opts := &client.ListOptions{LabelSelector: selector, Namespace: k6.NamespacedName().Namespace}
	jl := &batchv1.JobList{}
	var err error

	if err = r.List(ctx, jl, opts); err != nil {
		log.Error(err, "Could not list jobs")
		return
	}

	// TODO: We should distinguish between Suceeded/Failed/Unknown
	var finished int32
	for _, job := range jl.Items {
		if job.Status.Active != 0 {
			continue
		}
		finished++
	}

	log.Info(fmt.Sprintf("%d/%d jobs complete", finished, k6.GetSpec().Parallelism))

	if finished < k6.GetSpec().Parallelism {
		return
	}

	allFinished = true
	return
}
