package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/resources/jobs"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateJobs creates jobs that will spawn k6 pods for distributed test
func CreateJobs(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler) (ctrl.Result, error) {
	var (
		err   error
		res   ctrl.Result
		token string // only for cloud output tests
	)

	if len(k6.Status.TestRunID) > 0 {
		log = log.WithValues("testRunId", k6.Status.TestRunID)

		var (
			secrets    corev1.SecretList
			secretOpts = &client.ListOptions{
				// TODO: find out a better way to get namespace here
				Namespace: "k6-operator-system",
				LabelSelector: labels.SelectorFromSet(map[string]string{
					"k6cloud": "token",
				}),
			}
		)
		if err := r.List(ctx, &secrets, secretOpts); err != nil {
			log.Error(err, "Failed to load k6 Cloud token")
			return res, err
		}

		if len(secrets.Items) < 1 {
			err := fmt.Errorf("There are no secrets to hold k6 Cloud token")
			log.Error(err, err.Error())
			return res, err
		}

		if t, ok := secrets.Items[0].Data["token"]; !ok {
			err := fmt.Errorf("The secret doesn't have a field token for k6 Cloud")
			log.Error(err, err.Error())
			return res, err
		} else {
			token = string(t)
		}
		log.Info("Token for k6 Cloud was loaded.")
	}

	log.Info("Creating test jobs")

	if res, err = createJobSpecs(ctx, log, k6, r, token); err != nil {
		return res, err
	}

	log.Info("Changing stage of K6 status to created")
	k6.Status.Stage = "created"
	if err = r.Client.Status().Update(ctx, k6); err != nil {
		log.Error(err, "Could not update status of custom resource")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func createJobSpecs(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler, token string) (ctrl.Result, error) {
	found := &batchv1.Job{}
	namespacedName := types.NamespacedName{
		Name:      fmt.Sprintf("%s-1", k6.Name),
		Namespace: k6.Namespace,
	}

	if err := r.Get(ctx, namespacedName, found); err == nil || !errors.IsNotFound(err) {
		log.Info("Could not start a new test, Make sure you've deleted your previous run.")
		return ctrl.Result{}, err
	}

	for i := 1; i <= int(k6.Spec.Parallelism); i++ {
		if err := launchTest(ctx, k6, i, log, r, token); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func launchTest(ctx context.Context, k6 *v1alpha1.K6, index int, log logr.Logger, r *K6Reconciler, token string) error {
	var job *batchv1.Job
	var service *corev1.Service
	var err error

	msg := fmt.Sprintf("Launching k6 test #%d", index)
	log.Info(msg)

	if job, err = jobs.NewRunnerJob(k6, index, token); err != nil {
		log.Error(err, "Failed to generate k6 test job")
		return err
	}

	log.Info(fmt.Sprintf("Runner job is ready to start with image `%s` and command `%s`",
		job.Spec.Template.Spec.Containers[0].Image, job.Spec.Template.Spec.Containers[0].Command))

	if err = ctrl.SetControllerReference(k6, job, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference for job")
		return err
	}

	if err = r.Create(ctx, job); err != nil {
		log.Error(err, "Failed to launch k6 test")
		return err
	}

	if service, err = jobs.NewRunnerService(k6, index); err != nil {
		log.Error(err, "Failed to generate k6 test service")
		return err
	}

	if err = ctrl.SetControllerReference(k6, service, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference for service")
		return err
	}

	if err = r.Create(ctx, service); err != nil {
		log.Error(err, "Failed to launch k6 test services")
		return err
	}

	return nil
}
