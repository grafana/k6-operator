package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/grafana/k6-operator/pkg/types"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/resources/jobs"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	inspectOutput, inspectReady, err := inspectTestRun(ctx, log, *k6, r)
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

func SetupCloudTest(ctx context.Context, log logr.Logger, k6 *v1alpha1.K6, r *K6Reconciler) (res ctrl.Result, err error) {
	res = ctrl.Result{RequeueAfter: time.Second * 5}

	inspectOutput, inspectReady, err := inspectTestRun(ctx, log, *k6, r)
	if err != nil {
		// This *shouldn't* fail since it was already done once. Don't requeue.
		// Alternatively: store inspect options in K6 Status? Get rid off reading logs?
		return ctrl.Result{}, nil
	}
	if !inspectReady {
		return res, nil
	}

	token, tokenReady, err := loadToken(ctx, log, r)
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

// It may take some time to retrieve inspect output so indicate with boolean if it's ready
// and use returnErr only for errors that require a change of behaviour. All other errors
// should just be logged.
func inspectTestRun(ctx context.Context, log logr.Logger, k6 v1alpha1.K6, r *K6Reconciler) (
	inspectOutput cloud.InspectOutput, ready bool, returnErr error) {
	var (
		listOpts = &client.ListOptions{
			Namespace: k6.Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app":      "k6",
				"k6_cr":    k6.Name,
				"job-name": fmt.Sprintf("%s-initializer", k6.Name),
			}),
		}
		podList = &corev1.PodList{}
		err     error
	)
	if err = r.List(ctx, podList, listOpts); err != nil {
		log.Error(err, "Could not list pods")
		return
	}
	if len(podList.Items) < 1 {
		log.Info("No initializing pod found yet")
		return
	}

	// there should be only 1 initializer pod
	if podList.Items[0].Status.Phase != "Succeeded" {
		log.Info("Waiting for initializing pod to finish")
		return
	}

	// Here we need to get the output of the pod
	// pods/log is not currently supported by controller-runtime client and it is officially
	// recommended to use REST client instead:
	// https://github.com/kubernetes-sigs/controller-runtime/issues/1229

	// TODO: if the below errors repeat several times, it'd be a real error case scenario.
	// How likely is it? Should we track frequency of these errors here?
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err, "unable to fetch in-cluster REST config")
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err, "unable to get access to clientset")
		return
	}
	req := clientset.CoreV1().Pods(k6.Namespace).GetLogs(podList.Items[0].Name, &corev1.PodLogOptions{
		Container: "k6",
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	podLogs, err := req.Stream(ctx)
	if err != nil {
		log.Error(err, "unable to stream logs from the pod")
		return
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		log.Error(err, "unable to copy logs from the pod")
		return
	}

	if returnErr = json.Unmarshal(buf.Bytes(), &inspectOutput); returnErr != nil {
		// this shouldn't normally happen but if it does, let's log output by default
		log.Error(returnErr, fmt.Sprintf("unable to marshal: `%s`", buf.String()))
	}

	ready = true
	return
}

// Similarly to inspectTestRun, there may be some errors during load of token
// that should be just waited out. But other errors should result in change of
// behaviour in the caller.
// ready shows whether token was loaded yet, while returnErr indicates an error
// that should be acted on.
func loadToken(ctx context.Context, log logr.Logger, r *K6Reconciler) (token string, ready bool, returnErr error) {
	var (
		secrets    corev1.SecretList
		secretOpts = &client.ListOptions{
			// TODO: find out a better way to get namespace here
			Namespace: "k6-operator-system",
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"k6cloud": "token",
			}),
		}
		err error
	)

	if err = r.List(ctx, &secrets, secretOpts); err != nil {
		log.Error(err, "Failed to load k6 Cloud token")
		// This may be a networking issue, etc. so just retry.
		return
	}

	if len(secrets.Items) < 1 {
		// we should stop execution in case of this error
		returnErr = fmt.Errorf("There are no secrets to hold k6 Cloud token")
		log.Error(returnErr, err.Error())
		return
	}

	if t, ok := secrets.Items[0].Data["token"]; !ok {
		// we should stop execution in case of this error
		returnErr = fmt.Errorf("The secret doesn't have a field token for k6 Cloud")
		log.Error(err, err.Error())
		return
	} else {
		token = string(t)
		ready = true
		log.Info("Token for k6 Cloud was loaded.")
	}

	return
}

func getEnvVar(vars []corev1.EnvVar, name string) string {
	for _, v := range vars {
		if v.Name == name {
			return v.Value
		}
	}
	return ""
}
