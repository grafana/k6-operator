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
	"github.com/k6io/operator/pkg/resources"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/k6io/operator/api/v1alpha1"
)

// K6Reconciler reconciles a K6 object
type K6Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k6.io,resources=k6s,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k6.io,resources=k6s/status,verbs=get;update;patch

// +kubebuilder:rbac:groups=k6.io,resources=k6s,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k6.io,resources=k6s/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *K6Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("k6", req.NamespacedName)

	result := ctrl.Result{}

	// Fetch the applied CRD
	k6 := &v1alpha1.K6{}
	err := r.Get(ctx, req.NamespacedName, k6)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Request deleted. Skipping requeuing.")
			return result, nil
		}
		log.Error(err, "Could not fetch request")
		return ctrl.Result{Requeue: true}, err
	}

	// Check for previous jobs
	found := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-1", k6.Name), Namespace: k6.Namespace}, found)
	if err == nil || !errors.IsNotFound(err) {
		log.Info("Could not start a new test, Make sure you've deleted your previous run.")
		return result, err
	}

	// Create jobs
	for i := 1; i <= int(k6.Spec.Parallelism); i++ {
		log.Info(fmt.Sprintf("Launching k6 test #%d", i))
		job := resources.NewJob(k6, i)
		if err = ctrl.SetControllerReference(k6, job, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for job")
			return ctrl.Result{}, err
		}
		if err = r.Create(ctx, job); err != nil {
			log.Error(err, "Failed to launch k6 test")
			return ctrl.Result{}, err
		}
	}

	return result, nil
}

func (r *K6Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.K6{}).
		Complete(r)
}
