/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const k6CrLabelName = "k6_cr"

// K6Reconciler reconciles a K6 object
type K6Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile takes a K6 object and takes the appropriate action in the cluster
// +kubebuilder:rbac:groups=k6.io,resources=k6s,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k6.io,resources=k6s/status;k6s/finalizers,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods;pods/log,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *K6Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("namespace", req.Namespace, "name", req.Name, "reconcileID", controller.ReconcileIDFromContext(ctx))

	// Fetch the CRD
	k6 := &v1alpha1.K6{}
	err := r.Get(ctx, req.NamespacedName, k6)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("Request deleted. Nothing to reconcile.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Could not fetch request")
		return ctrl.Result{Requeue: true}, err
	}

	log.Info(fmt.Sprintf("Reconcile(); stage = %s", k6.Status.Stage))

	// Decision making here is now a mix between stages and conditions.
	// TODO: refactor further.

	switch k6.Status.Stage {
	case "":
		log.Info("Initialize test")

		k6.InitializeConditions()

		if _, err := r.UpdateStatus(ctx, k6, log); err != nil {
			return ctrl.Result{}, err
		}

		log.Info("Changing stage of K6 status to initialization")
		k6.Status.Stage = "initialization"
		if updateHappened, err := r.UpdateStatus(ctx, k6, log); err != nil {
			return ctrl.Result{}, err
		} else if updateHappened {
			return InitializeJobs(ctx, log, k6, r)
		}
		return ctrl.Result{}, nil

	case "initialization":
		if k6.IsUnknown(v1alpha1.CloudTestRun) {
			return RunValidations(ctx, log, k6, r)
		}

		if k6.IsFalse(v1alpha1.CloudTestRun) {
			// RunValidations has already happened and this is not a
			// cloud test: we can move on
			log.Info("Changing stage of K6 status to initialized")

			k6.Status.Stage = "initialized"

			if updateHappened, err := r.UpdateStatus(ctx, k6, log); err != nil {
				return ctrl.Result{}, err
			} else if updateHappened {
				return ctrl.Result{}, nil
			}
		}

		// log.Info(fmt.Sprintf("Debug \"initialization\" %v %v",
		// 	k6.IsTrue(v1alpha1.CloudTestRun),
		// 	k6.IsTrue(v1alpha1.CloudTestRunCreated)))

		if k6.IsTrue(v1alpha1.CloudTestRun) {

			if k6.IsFalse(v1alpha1.CloudTestRunCreated) {
				return SetupCloudTest(ctx, log, k6, r)

			} else {
				// if test run was created, then only changing status is left
				log.Info("Changing stage of K6 status to initialized")

				k6.Status.Stage = "initialized"

				if _, err := r.UpdateStatus(ctx, k6, log); err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		return ctrl.Result{}, nil

	case "initialized":
		return CreateJobs(ctx, log, k6, r)

	case "created":
		return StartJobs(ctx, log, k6, r)

	case "started":
		// log.Info(fmt.Sprintf("Debug \"started\" %v %v",
		// 	k6.IsTrue(v1alpha1.CloudTestRun),
		// 	k6.IsTrue(v1alpha1.CloudTestRunFinalized)))

		if k6.IsTrue(v1alpha1.CloudTestRun) && k6.IsTrue(v1alpha1.CloudTestRunFinalized) {
			// a fluke - nothing to do
			return ctrl.Result{}, nil
		}

		// wait for the test to finish
		if !FinishJobs(ctx, log, k6, r) {
			// Test runs can take a long time and usually they aren't supposed
			// to be too quick. So check in only periodically.
			return ctrl.Result{RequeueAfter: time.Second * 15}, nil
		}

		log.Info("All runner pods are finished")

		// now mark it as finished

		if k6.IsTrue(v1alpha1.TestRunRunning) {
			k6.UpdateCondition(v1alpha1.TestRunRunning, metav1.ConditionFalse)

			log.Info("Changing stage of K6 status to finished")
			k6.Status.Stage = "finished"

			// If this is a test run with cloud output, try to finalize it.
			if k6.IsTrue(v1alpha1.CloudTestRun) && k6.IsFalse(v1alpha1.CloudTestRunFinalized) {
				if err = cloud.FinishTestRun(k6.Status.TestRunID); err != nil {
					log.Error(err, "Failed to finalize the test run with cloud output")
					return ctrl.Result{}, nil
				} else {
					log.Info(fmt.Sprintf("Cloud test run %s was finalized succesfully", k6.Status.TestRunID))

					k6.UpdateCondition(v1alpha1.CloudTestRunFinalized, metav1.ConditionTrue)
				}
			}

			_, err := r.UpdateStatus(ctx, k6, log)
			if err != nil {
				return ctrl.Result{}, err
			}
			// log.Info(fmt.Sprintf("Debug updating status after finalize %v", updateHappened))
		}

		return ctrl.Result{}, nil

	case "error", "finished":
		// delete if configured
		if k6.Spec.Cleanup == "post" {
			log.Info("Cleaning up all resources")
			r.Delete(ctx, k6)
		}
		// notify if configured
		return ctrl.Result{}, nil
	}

	err = fmt.Errorf("invalid status")
	log.Error(err, "Invalid status for the k6 resource.")
	return ctrl.Result{}, err
}

// SetupWithManager sets up a managed controller that will reconcile all events for the K6 CRD
func (r *K6Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.K6{}).
		Owns(&batchv1.Job{}).
		Watches(&source.Kind{Type: &v1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(
				func(object client.Object) []reconcile.Request {
					pod := object.(*v1.Pod)
					k6CrName, ok := pod.GetLabels()[k6CrLabelName]
					if !ok {
						return nil
					}
					return []reconcile.Request{
						{NamespacedName: types.NamespacedName{
							Name:      k6CrName,
							Namespace: object.GetNamespace(),
						}}}
				}),
			builder.WithPredicates(predicate.NewPredicateFuncs(
				func(object client.Object) bool {
					pod := object.(*v1.Pod)
					_, ok := pod.GetLabels()[k6CrLabelName]
					if !ok {
						return false
					}
					return true
				}))).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
			// RateLimiter - ?
		}).
		Complete(r)
}

func (r *K6Reconciler) UpdateStatus(ctx context.Context, k6 *v1alpha1.K6, log logr.Logger) (updateHappened bool, err error) {
	proposedStatus := k6.Status

	// re-fetch resource
	err = r.Get(ctx, types.NamespacedName{Namespace: k6.Namespace, Name: k6.Name}, k6)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("Request deleted. No status to update.")
			return false, nil
		}
		log.Error(err, "Could not fetch request")
		return false, err
	}

	cleanObj := k6.DeepCopyObject().(client.Object)

	// Update only if it's truly a newer version of the resource
	// in comparison to the recently fetched resource.
	isNewer := k6.Status.SetIfNewer(proposedStatus)
	if !isNewer {
		return false, nil
	}

	err = r.Client.Status().Patch(ctx, k6, client.MergeFrom(cleanObj))

	// TODO: look into retry.RetryOnConflict(retry.DefaultRetry, func() error{...})
	// to have retries of failing update here, in case of conflicts;
	// with optional retry bool arg probably.

	// TODO: what if resource was deleted right before Patch?
	// Add a check for IsNotFound(err).

	if err != nil {
		log.Error(err, "Could not update status of custom resource")
		return false, err
	}

	return true, nil
}
