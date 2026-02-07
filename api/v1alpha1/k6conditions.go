package v1alpha1

import (
	"strings"
	"time"

	"github.com/grafana/k6-operator/pkg/types"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TestRunRunning indicates if the test run is currently running.
	// - if empty / Unknown, it's any stage before k6 resume (starter)
	// - if False, it's after all runners have finished successfully or with error
	// - if True, it's after successful starter but before all runners have finished
	TestRunRunning = "TestRunRunning"

	// TeardownExecuted indicates whether the `teardown()` has been executed on one of the runners.
	// This condition can be used only in PLZ test runs.
	TeardownExecuted = "TeardownExecuted"

	// CloudTestRun indicates if this test run is supposed to be a cloud test run
	// (i.e. with `--out cloud` option).
	// - if empty / Unknown, the type of test is unknown yet
	// - if False, it is not a cloud test run
	// - if True, it is a cloud test run
	CloudTestRun = "CloudTestRun"

	// CloudTestRunCreated indicates if k6 Cloud test run ID has been created for this test.
	// - if empty / Unknown, it's either a non-cloud test run or it is a cloud test run
	// that wasn't created yet
	// - if False, it is a cloud test run and it is yet to be created
	// - if True, it is a cloud test run and it has been created already
	CloudTestRunCreated = "CloudTestRunCreated"

	// CloudTestRunFinalized indicates if k6 Cloud test run has been finalized.
	// - if empty / Unknown, it's either a non-cloud test run or it is a cloud test run
	// that wasn't finalized yet
	// - if False, it's a cloud test run and it is yet to be finalized
	// - if True, it's a cloud test run that has been finalized already
	CloudTestRunFinalized = "CloudTestRunFinalized"

	// CloudPLZTestRun indicates if this k6 Cloud test run is a PLZ test run.
	// This condition is valid only if CloudTestRun is True as well.
	// - if empty / Unknown, it's either a non-PLZ test run or it's unknown yet.
	// - if False, it's not a PLZ test run.
	// - if True, it is a PLZ test run.
	CloudPLZTestRun = "CloudPLZTestRun"

	// CloudTestRunAborted indicates if this k6 Cloud test run was aborted externally,
	// for any reason.
	// This condition is valid only if CloudPLZTestRun is True as well.
	// - if empty / Unknown, it's either a non-PLZ test run or it's unknown yet.
	// - if False, it's a PLZ test run and it wasn't aborted.
	// - if True, it is a PLZ test run and it was aborted.
	CloudTestRunAborted = "CloudTestRunAborted"
)

// Initialize defines only conditions common to all test runs.
func Initialize(k6 *TestRun) {
	t := metav1.Now()
	k6.GetStatus().Conditions = []metav1.Condition{
		metav1.Condition{
			Type:               CloudTestRun,
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: t,
			Reason:             "TestRunTypeUnknown",
			Message:            "",
		},
		metav1.Condition{
			Type:               TestRunRunning,
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: t,
			Reason:             "TestRunPreparation",
			Message:            "",
		},
		metav1.Condition{
			Type:               TeardownExecuted,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: t,
			Reason:             "TeardownExecutedFalse",
			Message:            "",
		},
	}

	UpdateCondition(k6, CloudTestRunAborted, metav1.ConditionFalse)

	// PLZ test run case
	if len(k6.GetSpec().TestRunID) > 0 {
		UpdateCondition(k6, CloudPLZTestRun, metav1.ConditionTrue)
		UpdateCondition(k6, CloudTestRunCreated, metav1.ConditionTrue)
		UpdateCondition(k6, CloudTestRunFinalized, metav1.ConditionFalse)

		k6.GetStatus().TestRunID = k6.GetSpec().TestRunID
	} else {
		UpdateCondition(k6, CloudPLZTestRun, metav1.ConditionFalse)
		// PLZ test run can be defined only via spec.testRunId;
		// otherwise it's not a PLZ test run.
	}
}

func UpdateCondition(k6 *TestRun, conditionType string, conditionStatus metav1.ConditionStatus) {
	types.UpdateCondition(&k6.GetStatus().Conditions, conditionType, conditionStatus)
}

func IsTrue(k6 *TestRun, conditionType string) bool {
	return meta.IsStatusConditionTrue(k6.GetStatus().Conditions, conditionType)
}

func IsFalse(k6 *TestRun, conditionType string) bool {
	return meta.IsStatusConditionFalse(k6.GetStatus().Conditions, conditionType)
}

func IsUnknown(k6 *TestRun, conditionType string) bool {
	return !IsFalse(k6, conditionType) && !IsTrue(k6, conditionType)
}

func LastUpdate(k6 *TestRun, conditionType string) (time.Time, bool) {
	cond := meta.FindStatusCondition(k6.GetStatus().Conditions, conditionType)
	if cond != nil {
		return cond.LastTransitionTime.Time, true
	}
	return time.Now(), false
}

// SetIfNewer changes k6status only if changes in proposedStatus are consistent
// with the expected progression of a test run. If there were any acceptable
// changes proposed, it returns true.
func (k6status *TestRunStatus) SetIfNewer(proposedStatus TestRunStatus) (isNewer bool) {
	isNewer = types.SetIfNewer(&k6status.Conditions, proposedStatus.Conditions,
		func(proposedCondition metav1.Condition) (isNewer bool) {
			// Accept change of test run ID only if it's not set yet and together with
			// corresponding condition.
			if proposedCondition.Type == CloudTestRunCreated &&
				len(k6status.TestRunID) == 0 &&
				len(proposedStatus.TestRunID) > 0 {
				k6status.TestRunID = proposedStatus.TestRunID
				isNewer = true
			}
			// log if proposedStatus.TestRunID is empty here?

			// similarly with aggregation vars
			if len(proposedStatus.AggregationVars) > 0 && len(k6status.AggregationVars) == 0 {
				k6status.AggregationVars = proposedStatus.AggregationVars
			}

			return
		})

	// If a change in stage is proposed, confirm that it is consistent with
	// expected flow of any test run.
	if k6status.Stage != proposedStatus.Stage && len(proposedStatus.Stage) > 0 {
		switch k6status.Stage {
		case "", "initialization":
			k6status.Stage = proposedStatus.Stage
			isNewer = true

		case "initialized":
			if !strings.HasPrefix(string(proposedStatus.Stage), "init") {
				k6status.Stage = proposedStatus.Stage
				isNewer = true
			}
		case "created":
			if proposedStatus.Stage == "started" ||
				proposedStatus.Stage == "finished" ||
				proposedStatus.Stage == "error" ||
				proposedStatus.Stage == "stopped" {
				k6status.Stage = proposedStatus.Stage
				isNewer = true
			}
		case "started":
			if proposedStatus.Stage == "stopped" ||
				proposedStatus.Stage == "finished" ||
				proposedStatus.Stage == "error" {
				k6status.Stage = proposedStatus.Stage
				isNewer = true
			}
		case "stopped":
			if proposedStatus.Stage == "finished" ||
				proposedStatus.Stage == "error" {
				k6status.Stage = proposedStatus.Stage
				isNewer = true
			}
			// in case of finished or error stage, skip
		}
	}

	return
}
