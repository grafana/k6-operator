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
	"time"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	plzworkers "github.com/grafana/k6-operator/pkg/plz"
)

const (
	plzFinalizer     = "privateloadzones.k6.io/finalizer"
	plzUIDAnnotation = "privateloadzones.k6.io/plz-uid"
)

// PrivateLoadZoneReconciler reconciles a PrivateLoadZone object
type PrivateLoadZoneReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	workers plzworkers.PLZWorkers
}

//+kubebuilder:rbac:groups=k6.io,resources=privateloadzones,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k6.io,resources=privateloadzones/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k6.io,resources=privateloadzones/finalizers,verbs=get;update;patch

// Reconcile takes a PrivateLoadZone object and takes the appropriate action in the cluster
func (r *PrivateLoadZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("namespace", req.Namespace, "name", req.Name, "reconcileID", controller.ReconcileIDFromContext(ctx))

	plz := &v1alpha1.PrivateLoadZone{}
	err := r.Get(ctx, req.NamespacedName, plz)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("Request deleted. Nothing to reconcile.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Could not fetch request")
		return ctrl.Result{Requeue: true}, err
	}

	var worker *plzworkers.PLZWorker
	// skipping error, as currently the true state is judged by PLZ resource,
	// not by the state of the in-memory worker
	worker, _ = r.workers.GetWorker(plz.Name)

	if plz.DeletionTimestamp.IsZero() && (plz.IsUnknown(v1alpha1.PLZRegistered) || plz.IsFalse(v1alpha1.PLZRegistered)) {
		if controllerutil.ContainsFinalizer(plz, plzFinalizer) {
			// PLZ has been already registered so just update status accordingly

			plz.Initialize()
			plz.UpdateCondition(v1alpha1.PLZRegistered, metav1.ConditionTrue)

			if _, err := r.UpdateStatus(ctx, plz, logger); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			// This is the first reconcile: a PLZ worker should be created and registered
			token, proceed, result := r.loadToken(ctx, plz.Spec.Token, plz.Namespace, logger)
			if !proceed {
				return result, nil
			}

			worker = plzworkers.NewPLZWorker(plz, token, r.Client, r.Log)
			uid, err := worker.Register(ctx)
			if err != nil {
				return ctrl.Result{}, err
			}

			if err := r.workers.AddWorker(plz.Name, worker); err != nil {
				// An error here probably means a duplicate reconcile request. Switch to debug?
				logger.Error(err, "Trying to register an existing PLZ.")
				return ctrl.Result{}, nil
			}

			controllerutil.AddFinalizer(plz, plzFinalizer)
			plz.SetAnnotations(map[string]string{
				plzUIDAnnotation: uid,
			})

			if err := r.Update(ctx, plz); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		if !plz.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(plz, plzFinalizer) {
			// PLZ has been deleted.

			// worker can be nil in case of a "timely" restart of k6-operator
			if worker != nil {
				worker.StopFactory()
				worker.Deregister(ctx)

				controllerutil.RemoveFinalizer(plz, plzFinalizer)

				if err := r.Update(ctx, plz); err != nil {
					return ctrl.Result{}, err
				}

				r.workers.DeleteWorker(plz.Name)

				// nothing left to do
				return ctrl.Result{}, nil
			}
		}
	}

	if plz.IsTrue(v1alpha1.PLZRegistered) {
		if worker == nil {
			// if this is after restart of the k6-operator, the in-memory workers
			// might be null and should be constructed
			token, proceed, result := r.loadToken(ctx, plz.Spec.Token, plz.Namespace, logger)
			if !proceed {
				return result, nil
			}
			worker = plzworkers.NewPLZWorker(plz, token, r.Client, r.Log)

			if err := r.workers.AddWorker(plz.Name, worker); err != nil {
				// An error here probably means a duplicate reconcile request. Switch to debug?
				logger.Error(err, "Trying to register an existing PLZ.")
				return ctrl.Result{}, nil
			}

			// If worker is reconstructed after a restart, we should
			// re-check if it is scheduled for deletion.
			return ctrl.Result{Requeue: true}, nil
		}

		worker.StartFactory()
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrivateLoadZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PrivateLoadZone{}).
		Complete(r)
}

// UpdateStatus is now using similar logic to TestRunReconciler:
// see if it can / should be refactored.
func (r *PrivateLoadZoneReconciler) UpdateStatus(
	ctx context.Context,
	plz *v1alpha1.PrivateLoadZone,
	log logr.Logger) (updateHappened bool, err error) {

	proposedStatus := plz.Status

	// re-fetch resource
	err = r.Get(ctx, types.NamespacedName{Namespace: plz.Namespace, Name: plz.Name}, plz)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("Request deleted. No status to update.")
			return false, nil
		}
		log.Error(err, "Could not fetch request")
		return false, err
	}

	cleanObj := plz.DeepCopyObject().(client.Object)

	// Update only if it's truly a newer version of the resource
	// in comparison to the recently fetched resource.
	isNewer := plz.Status.SetIfNewer(proposedStatus)
	if !isNewer {
		return false, nil
	}

	err = r.Client.Status().Patch(ctx, plz, client.MergeFrom(cleanObj))

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

func (r *PrivateLoadZoneReconciler) loadToken(ctx context.Context, tokenName, ns string, logger logr.Logger) (
	token string, proceed bool, result ctrl.Result) {

	tokenInfo := cloud.NewTokenInfo(tokenName, ns)
	err := tokenInfo.Load(ctx, logger, r.Client)

	if err != nil {
		// An error here means a very likely mis-configuration of the token.
		logger.Error(err, "A problem while getting token.")
		return "", false, ctrl.Result{}
	}
	if !tokenInfo.Ready {
		return "", false, ctrl.Result{RequeueAfter: time.Second * 5}
	}
	return tokenInfo.Value(), true, ctrl.Result{}
}
