package plz

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/resources/containers"
	"github.com/grafana/k6-operator/pkg/testrun"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PLZWorker is an internal representation of PrivateLoadZone, which is regularly
// polling GCk6 and can (in the future) receive async updates of the state through the channel
type PLZWorker struct {
	plz   v1alpha1.PrivateLoadZone
	token string // needed for cloud logs

	poller   *cloud.TestRunPoller
	template *testrun.Template

	k8sClient client.Client
	logger    logr.Logger
}

// NewPLZWorker constructs a PLZWorker, create a template for test runs and creates a poller.
func NewPLZWorker(plz *v1alpha1.PrivateLoadZone, token string, k8sClient client.Client, logger logr.Logger) *PLZWorker {
	w := &PLZWorker{
		plz:       *plz,
		token:     token,
		k8sClient: k8sClient,
		logger:    logger.WithValues("namespace", plz.Namespace, "name", plz.Name),
	}

	w.createTemplate(plz)
	w.poller = cloud.NewTestRunPoller(cloud.ApiURL(cloud.K6CloudHost()), w.token, w.plz.Name, w.logger)

	return w
}

// Register PLZ with the Cloud.
func (w *PLZWorker) Register(ctx context.Context) (string, error) {
	uid, err := w.plz.Register(ctx, w.logger, w.poller.Client)
	if err != nil {
		return "", err
	}

	w.logger.Info(fmt.Sprintf("PLZ %s is registered with k6 Cloud.", w.plz.Name))

	return uid, nil
}

// Deregister PLZ with the Cloud.
func (w *PLZWorker) Deregister(ctx context.Context) {
	// Since resource is being deleted, there isn't much to do about
	// deregistration error here.
	_ = w.plz.Deregister(ctx, w.logger, w.poller.Client)

	w.logger.Info(fmt.Sprintf("PLZ %s is deregistered with k6 Cloud.", w.plz.Name))
}

// StartFactory starts a poller and starts to watch the channel for new test runs.
func (w *PLZWorker) StartFactory() {
	if w.poller != nil && !w.poller.IsPolling() {
		w.poller.Start()
		go func() {
			w.logger.Info("Started factory for PLZ test runs.")

			for testRunId := range w.poller.GetTestRuns() {
				w.handle(testRunId)
			}
			// TODO: a potential leak
		}()
		w.logger.Info("Started polling k6 Cloud for new test runs.")
	}
}

// StopFactory stops the poller
func (w *PLZWorker) StopFactory() {
	if w.poller != nil {
		w.poller.Stop()
	}
}

// createTemplate creates a default template, applicable for all PLZ test runs.
// The only fields set here are the ones common to all PLZ test runs.
func (w *PLZWorker) createTemplate(plz *v1alpha1.PrivateLoadZone) {
	volume := corev1.Volume{
		Name: "archive-volume",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      "archive-volume",
		MountPath: "/test",
	}

	w.template = &testrun.Template{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: plz.Namespace,
		},
		Spec: v1alpha1.TestRunSpec{
			Runner: v1alpha1.Pod{
				ImagePullSecrets:   plz.Spec.ImagePullSecrets,
				ServiceAccountName: plz.Spec.ServiceAccountName,
				NodeSelector:       plz.Spec.NodeSelector,
				Resources:          plz.Spec.Resources,
				Volumes: []corev1.Volume{
					volume,
				},
				VolumeMounts: []corev1.VolumeMount{
					volumeMount,
				},
				EnvFrom: plz.Spec.Config.ToEnvFromSource(),
			},
			Starter: v1alpha1.Pod{
				ServiceAccountName: plz.Spec.ServiceAccountName,
				NodeSelector:       plz.Spec.NodeSelector,
				ImagePullSecrets:   plz.Spec.ImagePullSecrets,
			},
			Script: v1alpha1.K6Script{
				LocalFile: "/test/archive.tar",
			},
			Separate: false,
			Cleanup:  v1alpha1.Cleanup("post"),

			Token: plz.Spec.Token,
		},
	}
}

// complete modifies tr with data from trData, which is specific for this test run.
func (w *PLZWorker) complete(tr *v1alpha1.TestRun, trData *cloud.TestRunData) {
	tr.Name = testrun.PLZTestName(trData.TestRunID())

	initContainer := containers.NewS3InitContainer(
		trData.ArchiveURL,
		"ghcr.io/grafana/k6-operator:latest-starter",
		tr.Spec.Runner.VolumeMounts[0],
	)

	envVars := append(trData.EnvVars(), corev1.EnvVar{
		Name:  "K6_CLOUD_HOST",
		Value: cloud.K6CloudHost(),
	})

	envVars = append(envVars, cloud.AggregationEnvVars(&trData.RuntimeConfig)...)

	tr.Spec.Runner.Image = trData.RunnerImage
	tr.Spec.Runner.InitContainers = []v1alpha1.InitContainer{
		initContainer,
	}
	tr.Spec.Runner.Env = envVars
	tr.Spec.Parallelism = int32(trData.InstanceCount)
	tr.Spec.Arguments = fmt.Sprintf(`--out cloud --no-thresholds --log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=%s,label.test_run_id=%s,header.Authorization="Token $(K6_CLOUD_TOKEN)"`,
		w.plz.Name,
		trData.TestRunID())
	tr.Spec.TestRunID = trData.TestRunID()
}

// handle creates a new PLZ TestRun from the given test run id
// TODO: pass proper context!
func (w *PLZWorker) handle(testRunId string) {
	tr := w.template.Create()

	// First check if such a test already exists
	namespacedName := types.NamespacedName{
		Name:      testrun.PLZTestName(testRunId),
		Namespace: tr.Namespace,
	}

	if err := w.k8sClient.Get(context.Background(), namespacedName, tr); err == nil || !errors.IsNotFound(err) {
		w.logger.Info(fmt.Sprintf("Test run `%s` has already been started.", testRunId))
		return
	}

	// Test does not exist so get its data and create it.

	trData, err := cloud.GetTestRunData(w.poller.Client, testRunId)
	if err != nil {
		w.logger.Error(err, fmt.Sprintf("Failed to retrieve test run data for `%s`", testRunId))
		return
	}
	w.complete(tr, trData)

	w.logger.Info(fmt.Sprintf("PLZ test run has been prepared with image `%s` and `%d` instances",
		tr.Spec.Runner.Image, tr.Spec.Parallelism), "testRunId", testRunId)

	if err := ctrl.SetControllerReference(&w.plz, tr, scheme); err != nil {
		w.logger.Error(err, "Failed to set controller reference for the PLZ test run", "testRunId", testRunId)
	}

	if err := w.k8sClient.Create(context.Background(), tr); err != nil {
		w.logger.Error(err, "Failed to create PLZ test run", "testRunId", testRunId)
	}

	w.logger.Info("Created new test run", "testRunId", testRunId)
}
