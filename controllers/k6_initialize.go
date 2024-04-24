package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/k6-operator/pkg/types"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/resources/jobs"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// InitializeJobs creates jobs that will run initial checks for distributed test if any are necessary
func InitializeJobs(ctx context.Context, log logr.Logger, k6 v1alpha1.TestRunI, r *TestRunReconciler) (res ctrl.Result, err error) {
	// initializer is a quick job so check in frequently
	res = ctrl.Result{RequeueAfter: time.Second * 5}

	cli := types.ParseCLI(k6.GetSpec().Arguments)

	var initializer *batchv1.Job
	if initializer, err = jobs.NewInitializerJob(k6, cli.ArchiveArgs); err != nil {
		return res, err
	}

	log.Info(fmt.Sprintf("Initializer job is ready to start with image `%s` and command `%s`",
		initializer.Spec.Template.Spec.Containers[0].Image, initializer.Spec.Template.Spec.Containers[0].Command))

	if err = ctrl.SetControllerReference(k6, initializer, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference for the initialize job")
		return res, err
	}

	if err = r.Create(ctx, initializer); err != nil {
		log.Error(err, "Failed to launch k6 test initializer")
		return res, err
	}

	return res, nil
}

func RunValidations(ctx context.Context, log logr.Logger, k6 v1alpha1.TestRunI, r *TestRunReconciler) (
	res ctrl.Result, ready bool, err error,
) {
	ready = true // TODO: to be removed
	rec := InitializerReconciler(r.Config)
	res, err = rec.Reconcile(ctx, k6)
	return
}

// SetupCloudTest inspects the output of initializer and creates a new
// test run. It is meant to be used only in cloud output mode.
func SetupCloudTest(ctx context.Context, log logr.Logger, k6 v1alpha1.TestRunI, r *TestRunReconciler) (res ctrl.Result, err error) {
	res = ctrl.Result{RequeueAfter: time.Second * 5}

	inspectOutput, inspectReady, err := inspectTestRun(ctx, log, k6, r.Client)
	if err != nil {
		// This *shouldn't* fail since it was already done once. Don't requeue.
		// Alternatively: store inspect options in K6 Status? Get rid off reading logs?
		return ctrl.Result{}, nil
	}
	if !inspectReady {
		return res, nil
	}

	token, tokenReady, err := loadToken(ctx, log, r.Client, "", nil)
	if err != nil {
		// An error here means a very likely mis-configuration of the token.
		// Consider updating status to error to let a user know quicker?
		log.Error(err, "A problem while getting token.")
		return ctrl.Result{}, nil
	}
	if !tokenReady {
		return res, nil
	}

	host := getEnvVar(k6.GetSpec().Runner.Env, "K6_CLOUD_HOST")

	if v1alpha1.IsFalse(k6, v1alpha1.CloudTestRunCreated) {

		// If CloudTestRunCreated has just been updated, wait for a bit before
		// acting, to avoid race condition between different reconcile loops.
		t, _ := v1alpha1.LastUpdate(k6, v1alpha1.CloudTestRunCreated)
		if time.Since(t) < 5*time.Second {
			return ctrl.Result{RequeueAfter: time.Second * 2}, nil
		}

		if len(inspectOutput.TestName()) < 1 {
			// script has already been parsed for initializer job definition,
			// so this is safe
			script, _ := k6.GetSpec().ParseScript()
			inspectOutput.SetTestName(script.Filename)
		}

		if testRunData, err := cloud.CreateTestRun(inspectOutput, k6.GetSpec().Parallelism, host, token, log); err != nil {
			log.Error(err, "Failed to create a new cloud test run.")
			return res, nil
		} else {
			log = log.WithValues("testRunId", testRunData.ReferenceID)
			log.Info(fmt.Sprintf("Created cloud test run: %s", testRunData.ReferenceID))

			k6.GetStatus().TestRunID = testRunData.ReferenceID
			v1alpha1.UpdateCondition(k6, v1alpha1.CloudTestRunCreated, metav1.ConditionTrue)

			k6.GetStatus().AggregationVars = cloud.EncodeAggregationConfig(testRunData.ConfigOverride)

			_, err := r.UpdateStatus(ctx, k6, log)
			// log.Info(fmt.Sprintf("Debug updating status after create %v", updateHappened))
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}
