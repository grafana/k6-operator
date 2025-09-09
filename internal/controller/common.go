package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/go-logr/logr"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/testrun"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errMessageTooLong = "Creation of %s takes too long: your configuration might be off. Check if %v were created successfully."
)

// It may take some time to retrieve inspect output so indicate with boolean if it's ready
// and use returnErr only for errors that require a change of behaviour. All other errors
// should just be logged.
func inspectTestRun(ctx context.Context, log logr.Logger, k6 *v1alpha1.TestRun, c client.Client) (
	inspectOutput cloud.InspectOutput, ready bool, returnErr error) {
	var (
		listOpts = &client.ListOptions{
			Namespace: k6.NamespacedName().Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app":      "k6",
				"k6_cr":    k6.NamespacedName().Name,
				"job-name": fmt.Sprintf("%s-initializer", k6.NamespacedName().Name),
			}),
		}
		podList = &corev1.PodList{}
		err     error
	)
	if err = c.List(ctx, podList, listOpts); err != nil {
		returnErr = err
		log.Error(err, "Could not list pods")
		return
	}

	if len(podList.Items) < 1 {
		log.Info("No initializing pod found yet")
		return
	}

	// there should be only 1 initializer pod
	if podList.Items[0].Status.Phase == corev1.PodFailed {
		returnErr = errors.New("initalizer job has failed")
		log.Error(returnErr, "error:")
		return
	}
	if podList.Items[0].Status.Phase != corev1.PodSucceeded {
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
		returnErr = err
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err, "unable to get access to clientset")
		returnErr = err
		return
	}
	req := clientset.CoreV1().Pods(k6.NamespacedName().Namespace).GetLogs(podList.Items[0].Name, &corev1.PodLogOptions{
		Container: "k6",
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	podLogs, err := req.Stream(ctx)
	if err != nil {
		log.Error(err, "unable to stream logs from the pod")
		returnErr = err
		return
	}
	defer podLogs.Close() //nolint:errcheck

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		log.Error(err, "unable to copy logs from the pod")
		returnErr = err
		return
	}

	if returnErr = json.Unmarshal(buf.Bytes(), &inspectOutput); returnErr != nil {
		// this shouldn't normally happen but if it does, let's log output by default
		log.Error(returnErr, fmt.Sprintf("unable to marshal: `%s`", buf.String()))
	}

	ready = true
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

func (r *TestRunReconciler) hostnames(ctx context.Context, log logr.Logger, abortOnUnready bool, opts *client.ListOptions) ([]string, error) {
	var (
		hostnames []string
		err       error
	)

	sl := &corev1.ServiceList{}

	if err = r.List(ctx, sl, opts); err != nil {
		log.Error(err, "Could not list services")
		return nil, err
	}

	for _, service := range sl.Items {
		log.Info(fmt.Sprintf("Checking service %s", service.Name))
		if isServiceReady(log, &service) {
			log.Info(fmt.Sprintf("%v service is ready", service.Name))
			hostnames = append(hostnames, service.Spec.ClusterIP)
		} else {
			err = fmt.Errorf("%v service is not ready", service.Name)
			log.Info(err.Error())
			if abortOnUnready {
				return nil, err
			}
		}
	}

	return hostnames, nil
}

func runSetup(ctx context.Context, hostnames []string, log logr.Logger) error {
	log.Info("Invoking setup() on the first runner")

	setupData, err := testrun.RunSetup(ctx, hostnames[0])
	if err != nil {
		return err
	}

	log.Info("Sending setup data to the runners")

	if err = testrun.SetSetupData(ctx, hostnames, setupData); err != nil {
		return err
	}

	return nil
}

func runTeardown(ctx context.Context, hostnames []string, log logr.Logger) {
	log.Info("Invoking teardown() on the first responsive runner")

	if err := testrun.RunTeardown(ctx, hostnames); err != nil {
		log.Error(err, "Failed to invoke teardown()")
	}
}
