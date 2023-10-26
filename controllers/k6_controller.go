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

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/grafana/k6-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	k6CrLabelName = "k6_cr"
)

type K6Reconciler struct {
	t *TestRunReconciler
}

func NewK6Reconciler(t *TestRunReconciler) *K6Reconciler {
	return &K6Reconciler{t}
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
	log := r.t.Log.WithValues("namespace", req.Namespace, "name", req.Name, "reconcileID", controller.ReconcileIDFromContext(ctx))

	// Fetch the CRD
	k6 := &v1alpha1.K6{}
	err := r.t.Get(ctx, req.NamespacedName, k6)

	if err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("Request deleted. Nothing to reconcile.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Could not fetch request")
		return ctrl.Result{Requeue: true}, err
	}

	return r.t.reconcile(ctx, req, log, k6)
}

// SetupWithManager sets up a managed controller that will reconcile all events for the K6 CRD
func (r *K6Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.K6{}).
		Owns(&batchv1.Job{}).
		Watches(&v1.Pod{},
			handler.EnqueueRequestsFromMapFunc(
				func(ctx context.Context, object client.Object) []reconcile.Request {
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
					return ok
				}))).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
			// RateLimiter - ?
		}).
		Complete(r)
}
