package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"go.k6.io/k6/v2/cloudapi"
	"go.k6.io/k6/v2/errext/exitcodes"
	corev1 "k8s.io/api/core/v1"
)

func sendCloudError(
	ctx context.Context,
	log logr.Logger,
	k6 *v1alpha1.TestRun,
	cloudClient *cloudapi.Client,
	code cloud.ErrorCode,
	detail string,
) {
	if v1alpha1.IsTrue(k6, v1alpha1.CloudPLZTestRun) {
		notifyPLZ(ctx, log, k6, cloudClient, &cloud.TestRunNotification{
			Code:   code,
			Reason: detail,
		})
		return
	}

	if v1alpha1.IsTrue(k6, v1alpha1.CloudTestRun) {
		events := cloud.ErrorEvent(code).
			WithDetail(detail).
			WithAbort()
		cloud.SendTestRunEvents(cloudClient, k6.TestRunID(), log, events)
	}
}

func finishCloudTestRun(ctx context.Context, k6 *v1alpha1.TestRun, cloudClient *cloudapi.Client) error {
	if v1alpha1.IsTrue(k6, v1alpha1.CloudPLZTestRun) {
		return cloud.NotifyTestRun(ctx, cloudClient, plzToken(k6), k6.TestRunID(), nil)
	}
	return cloud.FinishTestRun(cloudClient, k6.GetStatus().TestRunID)
}

func notifyPLZ(
	ctx context.Context,
	log logr.Logger,
	k6 *v1alpha1.TestRun,
	cloudClient *cloudapi.Client,
	executionError *cloud.TestRunNotification,
) {
	if err := cloud.NotifyTestRun(ctx, cloudClient, plzToken(k6), k6.TestRunID(), executionError); err != nil {
		log.Error(err, "Failed to notify k6 Cloud", "testRunId", k6.TestRunID())
	}
}

func plzToken(k6 *v1alpha1.TestRun) string {
	return getEnvVar(k6.GetSpec().Runner.Env, "K6_CLOUD_TOKEN")
}

// runnerExitError only observes containers that have exited; --linger can hide
// runtime results until cleanup.
// beforeExecution distinguishes whether runnerExitError was called before `main` execution or after.
// OOM and timeout detection are TODOs.
func runnerExitError(pod corev1.Pod, beforeExecution bool) *cloud.TestRunNotification {
	for _, status := range pod.Status.ContainerStatuses {
		terminated := status.State.Terminated
		if status.Name != "k6" || terminated == nil || terminated.ExitCode == 0 {
			continue
		}
		detail := fmt.Sprintf("runner pod %s terminated with exit code %d: %s", pod.Name, terminated.ExitCode, terminated.Message)
		if beforeExecution {
			return &cloud.TestRunNotification{Code: cloud.K6OperatorStartError, Reason: "Runner failed before execution started", Detail: detail}
		}
		return runnerExecutionError(int(terminated.ExitCode), detail)
	}
	return nil
}

func runnerExecutionError(exitCode int, detail string) *cloud.TestRunNotification {
	var code cloud.ErrorCode
	var reason string
	switch exitCode {
	case int(exitcodes.ScriptException):
		code, reason = cloud.ScriptException, "Script execution failed"
	case int(exitcodes.ScriptAborted):
		code, reason = cloud.ScriptAborted, "Script was aborted"
	case int(exitcodes.GoPanic):
		code, reason = cloud.UnknownError, "k6 exited unexpectedly"
	case int(exitcodes.CannotStartRESTAPI), int(exitcodes.InvalidConfig):
		code, reason = cloud.K6OperatorStartError, "Runner failed to start"
	case int(exitcodes.ExternalAbort), -1:
		code, reason = cloud.K6OperatorRunnerError, "Runner failed during execution"
	default:
		return nil
	}
	return &cloud.TestRunNotification{Code: code, Reason: reason, Detail: detail}
}
