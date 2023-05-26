package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// It may take some time to retrieve inspect output so indicate with boolean if it's ready
// and use returnErr only for errors that require a change of behaviour. All other errors
// should just be logged.
func inspectTestRun(ctx context.Context, log logr.Logger, k6 v1alpha1.K6, c client.Client) (
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
	if err = c.List(ctx, podList, listOpts); err != nil {
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
//
// Currently we support two modes for loading token:
// - by name of the token (PLZ mode)
// - by label selector (cloud output mode)
// Specify arguments to loadToken accordingly.
func loadToken(ctx context.Context, log logr.Logger, c client.Client, secretName string, sOpts *client.ListOptions) (token string, ready bool, returnErr error) {
	var (
		secrets corev1.SecretList
		secret  corev1.Secret
		// This is the default location of the token;
		// what is used by cloud output mode.
		secretOpts = &client.ListOptions{
			Namespace: "k6-operator-system",
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"k6cloud": "token",
			}),
		}
		err error
	)

	if sOpts != nil {
		secretOpts = sOpts
	}

	if len(secretName) > 0 {
		log.Info("Loading token by name.", "name", secretName, "secretNamespace", sOpts.Namespace)

		if err := c.Get(ctx, types.NamespacedName{Namespace: sOpts.Namespace, Name: secretName}, &secret); err != nil {
			log.Error(err, "Failed to load k6 Cloud token")
			// This may be a networking issue, etc. so just retry.
			return
		}
	} else {
		log.Info("Loading token by label pair.", "labelset", secretOpts.LabelSelector.String(), "secretNamespace", secretOpts.Namespace)

		if err = c.List(ctx, &secrets, secretOpts); err != nil {
			log.Error(err, "Failed to load k6 Cloud token")
			// This may be a networking issue, etc. so just retry.
			return
		}

		if len(secrets.Items) < 1 {
			// we should stop execution in case of this error
			returnErr = fmt.Errorf("There are no secrets to hold k6 Cloud token")
			log.Error(returnErr, returnErr.Error())
			return
		}

		secret = secrets.Items[0]
	}

	if t, ok := secret.Data["token"]; !ok {
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
