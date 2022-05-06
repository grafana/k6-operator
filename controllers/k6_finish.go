package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FinishJobs waits for the pods to finish, performs finishing call for cloud output and moves state to "finished".
func FinishJobs(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler) (ctrl.Result, error) {
	log.Info("Waiting for pods to finish")

	// Here we assume that the test runs for some time and there is no need to
	// check it more often than twice in a minute.
	//
	// The total timeout for the test is set to duration of the test + 2 min.
	// These 2 min are meant to cover the time needed to start the pods: sometimes
	// pods are ready a bit later than operator reaches this stage so from the
	// viewpoint of operator it takes longer. This behaviour depends on the setup of
	// cluster. 2 min are meant to be a sufficient safeguard for such cases.

	testDuration := inspectOutput.TotalDuration.TimeDuration()

	err := wait.PollImmediate(time.Second*30, testDuration+time.Minute*2, func() (done bool, err error) {
		selector := labels.SelectorFromSet(map[string]string{
			"app":    "k6",
			"k6_cr":  k6.Name,
			"runner": "true",
		})

		opts := &client.ListOptions{LabelSelector: selector, Namespace: k6.Namespace}
		jl := &batchv1.JobList{}

		if err := r.List(ctx, jl, opts); err != nil {
			log.Error(err, "Could not list jobs")
			return false, nil
		}

		// TODO: We should distinguish between Suceeded/Failed/Unknown
		var finished int32
		for _, job := range jl.Items {
			if job.Status.Active != 0 {
				continue
			}
			finished++
		}

		log.Info(fmt.Sprintf("%d/%d jobs complete", finished, k6.Spec.Parallelism))

		if finished >= k6.Spec.Parallelism {
			return true, nil
		}

		return false, nil
	})

	if err != nil {
		log.Error(err, "Waiting for pods to finish ended with error")
	}

	// If this is a test run with cloud output, try to finalize it regardless.
	if cli := types.ParseCLI(&k6.Spec); cli.HasCloudOut {
		if err = cloud.FinishTestRun(testRunId); err != nil {
			log.Error(err, "Could not finalize the test run with cloud output")
		} else {
			log.Info(fmt.Sprintf("Cloud test run %s was finalized succesfully", testRunId))
		}
	}

	log.Info("Changing stage of K6 status to finished")
	k6.Status.Stage = "finished"
	if err = r.Client.Status().Update(ctx, k6); err != nil {
		log.Error(err, "Could not update status of custom resource")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
