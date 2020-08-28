package controllers

import (
	"context"
	"github.com/go-logr/logr"
	testv1alpha1 "github.com/k6io/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K6Reconciler reconciles a K6 object
type K6Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=test.k6.io,resources=k6s,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.k6.io,resources=k6s/status,verbs=get;update;patch

func (r *K6Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("k6", req.NamespacedName)

	k6 := &testv1alpha1.K6{}
	err := r.Get(ctx, req.NamespacedName, k6)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Request deleted. Skipping requeuing.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Could not fetch request")
		return ctrl.Result{Requeue: true}, err
	}

	found := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: k6.Name, Namespace: k6.Namespace}, found)
	if err == nil || !errors.IsNotFound(err) {
		log.Info("Could not start a new test, Make sure you've deleted your previous run.")
		return ctrl.Result{}, err
	}

	log.Info("Launching k6 test")
	deployment := r.newDeployment(k6)
	err = r.Create(ctx, deployment)
	if err != nil {
		log.Error(err, "Failed to launch k6 test")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *K6Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1alpha1.K6{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *K6Reconciler) newDeployment(k *testv1alpha1.K6) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.Name,
			Namespace: k.Namespace,
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &k.Spec.Nodes,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: newLabels(k.Name),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Image:   "loadimpact/k6:latest",
						Name:    "k6",
						Command: []string{"k6", "run", "/test/test.js"},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "k6-test-volume",
							MountPath: "/test",
						}},
						Env: []corev1.EnvVar{{
							Name:  "K6_NODES_INDEX",
							Value: "1",
						}, {
							Name:  "K6_NODES_TOTAL",
							Value: "1",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "k6-test-volume",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: k.Spec.Script,
								},
							},
						},
					}},
				},
			},
		},
	}
	err := ctrl.SetControllerReference(k, job, r.Scheme)
	if err != nil {
		return nil
	}
	return job
}

func newLabels(name string) map[string]string {
	return map[string]string{
		"app":   "k6",
		"k6_cr": name,
	}
}
