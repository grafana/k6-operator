package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/resources/jobs"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func isServiceReady(log logr.Logger, service *v1.Service) bool {
	resp, err := http.Get(fmt.Sprintf("http://%v:6565/v1/status", service.Spec.ClusterIP))

	if err != nil {
		log.Error(err, fmt.Sprintf("failed to get status from %v", service.ObjectMeta.Name))
		return false
	}

	return resp.StatusCode < 400
}

// StartJobs in the Ready phase using a curl container
func StartJobs(ctx context.Context, log logr.Logger, k6 *v1alpha1.TestRun, r *TestRunReconciler) (res ctrl.Result, err error) {
	// It may take some time to get Services up, so check in frequently
	res = ctrl.Result{RequeueAfter: time.Second}

	if len(k6.GetStatus().TestRunID) > 0 {
		log = log.WithValues("testRunId", k6.GetStatus().TestRunID)
	}

	log.Info("Waiting for pods to get ready")

	opts := k6.ListOptions()

	pl := &v1.PodList{}
	if err = r.List(ctx, pl, opts); err != nil {
		log.Error(err, "Could not list pods")
		return res, nil
	}

	var count int
	for _, pod := range pl.Items {
		if pod.Status.Phase != "Running" {
			continue
		}
		count++
	}

	log.Info(fmt.Sprintf("%d/%d runner pods ready", count, k6.GetSpec().Parallelism))

	if count != int(k6.GetSpec().Parallelism) {
		if t, ok := v1alpha1.LastUpdate(k6, v1alpha1.TestRunRunning); !ok {
			// this should never happen
			return res, errors.New("Cannot find condition TestRunRunning")
		} else {
			// let's try this approach
			if time.Since(t).Minutes() > 5 {
				msg := fmt.Sprintf(errMessageTooLong, "runner pods", "runner jobs and pods")
				log.Info(msg)

				if v1alpha1.IsTrue(k6, v1alpha1.CloudTestRun) {
					events := cloud.ErrorEvent(cloud.K6OperatorStartError).
						WithDetail(msg).
						WithAbort()
					cloud.SendTestRunEvents(r.k6CloudClient, k6.TestRunID(), log, events)
				}
			}
		}

		return res, nil
	}

	// services

	log.Info("Waiting for services to get ready")

	hostnames, err := r.hostnames(ctx, log, true, opts)
	log.Info(fmt.Sprintf("err: %v, hostnames: %v", err, hostnames))
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info(fmt.Sprintf("%d/%d services ready", len(hostnames), k6.GetSpec().Parallelism))

	// setup

	if v1alpha1.IsTrue(k6, v1alpha1.CloudPLZTestRun) {
		if err := runSetup(ctx, hostnames, log); err != nil {
			return ctrl.Result{}, err
		}
	}

	// starter
	
	// Handle Linux command line length limitation (max ~500 IPs)
	// Split hostnames into batches of 500 to avoid "argument list too long" error
	const maxHostnamesPerBatch = 500
	var hostnamesBatches [][]string
	
	// Split hostnames into batches
	for i := 0; i < len(hostnames); i += maxHostnamesPerBatch {
		end := i + maxHostnamesPerBatch
		if end > len(hostnames) {
			end = len(hostnames)
		}
		hostnamesBatches = append(hostnamesBatches, hostnames[i:end])
	}
	
	// Create a starter job for each batch
	jobsCreated := 0
	for batchIndex, batch := range hostnamesBatches {
		// Create a starter job for this batch
		starter := jobs.NewStarterJob(k6, batch)
		
		// Set a unique name for each starter job if there are multiple batches
		if len(hostnamesBatches) > 1 {
			starter.ObjectMeta.Name = fmt.Sprintf("%s-batch-%d", starter.ObjectMeta.Name, batchIndex+1)
		}
		
		if err = ctrl.SetControllerReference(k6, starter, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for the start job", "batch", batchIndex+1)
			continue
		}
		
		if err = r.Create(ctx, starter); err != nil {
			log.Error(err, "Failed to launch k6 test starter", "batch", batchIndex+1)
			continue
		}
		
		jobsCreated++
		log.Info("Created starter job", "batch", batchIndex+1, "hostnames", len(batch))
	}
	
	if jobsCreated == 0 {
		log.Error(fmt.Errorf("no starter jobs created"), "Failed to create any starter jobs")
		return res, nil
	}
	
	if len(hostnamesBatches) > 1 {
		log.Info(fmt.Sprintf("Created %d starter jobs to handle %d hostnames", jobsCreated, len(hostnames)))
	}

	log.Info("Changing stage of TestRun status to started")
	k6.GetStatus().Stage = "started"
	v1alpha1.UpdateCondition(k6, v1alpha1.TestRunRunning, metav1.ConditionTrue)

	if updateHappened, err := r.UpdateStatus(ctx, k6, log); err != nil {
		return ctrl.Result{}, err
	} else if updateHappened {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}
