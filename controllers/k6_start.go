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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func isPodReady(pod *v1.Pod) bool {
        if len(pod.Status.HostIP) < 1 {
            return false
        }
	resp, err := http.Get(fmt.Sprintf("%v/v1/status", pod.Status.HostIP))

	if err != nil {
		return false
	}

	return resp.StatusCode < 400
}

// StartJobs in the Ready phase using a curl container
func StartJobs(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler) (ctrl.Result, error) {
	log.Info("Waiting for pods to get ready")

	err := wait.PollImmediate(time.Second*5, time.Second*60, func() (done bool, err error) {
		selector := labels.SelectorFromSet(map[string]string{
			"app":   "k6",
			"k6_cr": k6.Name,
		})

		opts := &client.ListOptions{LabelSelector: selector, Namespace: k6.Namespace}
		pl := &v1.PodList{}

		if e := r.List(ctx, pl, opts); e != nil {
			log.Error(e, "Could not list pods")
			return false, e
		}

		var count int
		for _, pod := range pl.Items {
			if pod.Status.Phase != "Running" && !isPodReady(&pod) {
				continue
			}
			count++
		}

		log.Info(fmt.Sprintf("%d/%d pods ready", count, k6.Spec.Parallelism))

		if count != int(k6.Spec.Parallelism) {
			return false, nil
		}

		var hostnames []string

		sl := &v1.ServiceList{}

		if e := r.List(ctx, sl, opts); e != nil {
			log.Error(e, "Could not list services")
			return false, e
		}

		for _, service := range sl.Items {
			hostnames = append(hostnames, service.ObjectMeta.Name)
		}

		starter := jobs.NewStarterJob(k6, hostnames)

		if err = ctrl.SetControllerReference(k6, starter, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for job")
		}

		if err = r.Create(ctx, starter); err != nil {
			log.Error(err, "Failed to launch k6 test starter")
			return true, err
		}

		return true, nil
	})

	if err != nil {
		log.Error(err, "Failed to start all jobs")
		return ctrl.Result{}, err
	}

	k6.Status.Stage = "started"
	if err = r.Client.Status().Update(ctx, k6); err != nil {
		log.Error(err, "Could not update status of custom resource")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
