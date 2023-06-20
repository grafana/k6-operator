package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/resources/jobs"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func isServiceReady(log logr.Logger, service *v1.Service) bool {
	resp, err := http.Get(fmt.Sprintf("http://%v.%v.svc.cluster.local:6565/v1/status", service.ObjectMeta.Name, service.ObjectMeta.Namespace))

	if err != nil {
		log.Error(err, fmt.Sprintf("failed to get status from %v", service.ObjectMeta.Name))
		return false
	}

	return resp.StatusCode < 400
}

// StartJobs in the Ready phase using a curl container
func StartJobs(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler) (res ctrl.Result, err error) {
	// It may take some time to get Services up, so check in frequently
	res = ctrl.Result{RequeueAfter: time.Second}

	if len(k6.Status.TestRunID) > 0 {
		log = log.WithValues("testRunId", k6.Status.TestRunID)
	}

	log.Info("Waiting for pods to get ready")

	selector := labels.SelectorFromSet(map[string]string{
		"app":    "k6",
		"k6_cr":  k6.Name,
		"runner": "true",
	})

	opts := &client.ListOptions{LabelSelector: selector, Namespace: k6.Namespace}
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

	log.Info(fmt.Sprintf("%d/%d runner pods ready", count, k6.Spec.Parallelism))

	if count != int(k6.Spec.Parallelism) {
		return res, nil
	}

	var hostnames []string
	sl := &v1.ServiceList{}

	if err = r.List(ctx, sl, opts); err != nil {
		log.Error(err, "Could not list services")
		return res, nil
	}

	for _, service := range sl.Items {
		hostnames = append(hostnames, service.Spec.ClusterIP)

		if !isServiceReady(log, &service) {
			log.Info(fmt.Sprintf("%v service is not ready, aborting", service.ObjectMeta.Name))
			return res, nil
		} else {
			log.Info(fmt.Sprintf("%v service is ready", service.ObjectMeta.Name))
		}
	}

	starter := jobs.NewStarterJob(k6, hostnames)

	if err = ctrl.SetControllerReference(k6, starter, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference for the start job")
	}

	// TODO: add a check for existence of starter job

	if err = r.Create(ctx, starter); err != nil {
		log.Error(err, "Failed to launch k6 test starter")
		return res, nil
	}

	log.Info("Created starter job")

	log.Info("Changing stage of K6 status to started")
	k6.Status.Stage = "started"
	k6.UpdateCondition(v1alpha1.TestRunRunning, metav1.ConditionTrue)

	if updateHappened, err := r.UpdateStatus(ctx, k6, log); err != nil {
		return ctrl.Result{}, err
	} else if updateHappened {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}
