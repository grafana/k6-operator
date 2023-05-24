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
func InitializeJobs(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler) (res ctrl.Result, err error) {
	// initializer is a quick job so check in frequently
	res = ctrl.Result{RequeueAfter: time.Second * 5}

	cli := types.ParseCLI(&k6.Spec)

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

func RunValidations(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler) (res ctrl.Result, err error) {
	// initializer is a quick job so check in frequently
	res = ctrl.Result{RequeueAfter: time.Second * 5}

	cli := types.ParseCLI(&k6.Spec)

	inspectOutput, inspectReady, err := inspectTestRun(ctx, log, *k6, r.Client)
	if err != nil {
		// inspectTestRun made a log message already so just return without requeue
		return ctrl.Result{}, nil
	}
	if !inspectReady {
		return res, nil
	}

	log.Info(fmt.Sprintf("k6 inspect: %+v", inspectOutput))

	if int32(inspectOutput.MaxVUs) < k6.Spec.Parallelism {
		err = fmt.Errorf("number of instances > number of VUs")
		// TODO maybe change this to a warning and simply set parallelism = maxVUs and proceed with execution?
		// But logr doesn't seem to have warning level by default, only with V() method...
		// It makes sense to return to this after / during logr VS logrus issue https://github.com/grafana/k6-operator/issues/84
		log.Error(err, "Parallelism argument cannot be larger than maximum VUs in the script",
			"maxVUs", inspectOutput.MaxVUs,
			"parallelism", k6.Spec.Parallelism)

		k6.Status.Stage = "error"

		if _, err := r.UpdateStatus(ctx, k6, log); err != nil {
			return ctrl.Result{}, err
		}

		// Don't requeue in case of this error; unless it's made into a warning as described above.
		return ctrl.Result{}, nil
	}

	if cli.HasCloudOut {
		k6.UpdateCondition(v1alpha1.CloudTestRun, metav1.ConditionTrue)
		k6.UpdateCondition(v1alpha1.CloudTestRunCreated, metav1.ConditionFalse)
		k6.UpdateCondition(v1alpha1.CloudTestRunFinalized, metav1.ConditionFalse)
	} else {
		k6.UpdateCondition(v1alpha1.CloudTestRun, metav1.ConditionFalse)
	}

	if _, err := r.UpdateStatus(ctx, k6, log); err != nil {
		return ctrl.Result{}, err
	}

	return res, nil
}

// SetupCloudTest inspects the output of initializer and creates a new
// test run. It is meant to be used only in cloud output mode.
func SetupCloudTest(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler) (res ctrl.Result, err error) {
	res = ctrl.Result{RequeueAfter: time.Second * 5}

	inspectOutput, inspectReady, err := inspectTestRun(ctx, log, *k6, r.Client)
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

	host := getEnvVar(k6.Spec.Runner.Env, "K6_CLOUD_HOST")

	if k6.IsFalse(v1alpha1.CloudTestRunCreated) {

		// If CloudTestRunCreated has just been updated, wait for a bit before
		// acting, to avoid race condition between different reconcile loops.
		t, _ := k6.LastUpdate(v1alpha1.CloudTestRunCreated)
		if time.Now().Sub(t) < 5*time.Second {
			return ctrl.Result{RequeueAfter: time.Second * 2}, nil
		}

		if testRunData, err := cloud.CreateTestRun(inspectOutput, k6.Spec.Parallelism, host, token, log); err != nil {
			log.Error(err, "Failed to create a new cloud test run.")
			return res, nil
		} else {
			log = log.WithValues("testRunId", testRunData.ReferenceID)
			log.Info(fmt.Sprintf("Created cloud test run: %s", testRunData.ReferenceID))

			k6.Status.TestRunID = testRunData.ReferenceID
			k6.UpdateCondition(v1alpha1.CloudTestRunCreated, metav1.ConditionTrue)

			k6.Status.AggregationVars = cloud.EncodeAggregationConfig(testRunData)

			_, err := r.UpdateStatus(ctx, k6, log)
			// log.Info(fmt.Sprintf("Debug updating status after create %v", updateHappened))
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}
