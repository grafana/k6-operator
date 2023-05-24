package v1alpha1

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TestRunRunning indicates if the test run is currently running.
	// - if empty / Unknown, it's any stage before k6 resume (starter)
	// - if False, it's after all runners have finished successfully or with error
	// - if True, it's after successful starter but before all runners have finished
	TestRunRunning = "TestRunRunning"

	// CloudTestRun indicates if this test run is supposed to be a cloud test run.
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
)

var reasons = map[string]string{
	"TestRunRunningUnknown": "TestRunPreparation",
	"TestRunRunningTrue":    "TestRunRunningTrue",
	"TestRunRunningFalse":   "TestRunRunningFalse",

	"CloudTestRunUnknown": "TestRunTypeUnknown",
	"CloudTestRunTrue":    "CloudTestRunTrue",
	"CloudTestRunFalse":   "CloudTestRunFalse",

	"CloudTestRunCreatedUnknown": "CloudTestRunCreatedUnknown",
	"CloudTestRunCreatedTrue":    "CloudTestRunCreatedTrue",
	"CloudTestRunCreatedFalse":   "CloudTestRunCreatedFalse",

	"CloudTestRunFinalizedUnknown": "CloudTestRunFinalizedUnknown",
	"CloudTestRunFinalizedTrue":    "CloudTestRunFinalizedTrue",
	"CloudTestRunFinalizedFalse":   "CloudTestRunFinalizedFalse",

	"CloudPLZTestRunUnknown": "CloudPLZTestRunUnknown",
	"CloudPLZTestRunTrue":    "CloudPLZTestRunTrue",
	"CloudPLZTestRunFalse":   "CloudPLZTestRunFalse",

	"PLZRegisteredUnknown": "PLZRegisteredUnknown",
	"PLZRegisteredTrue":    "PLZRegisteredTrue",
	"PLZRegisteredFalse":   "PLZRegisteredFalse",
}

// Initialize defines only conditions common to all test runs.
func (k6 *K6) Initialize() {
	t := metav1.Now()
	k6.Status.Conditions = []metav1.Condition{
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
	}

	// PLZ test run case
	if len(k6.Spec.TestRunID) > 0 {
		k6.UpdateCondition(CloudTestRun, metav1.ConditionTrue)
		k6.UpdateCondition(CloudPLZTestRun, metav1.ConditionTrue)
		k6.UpdateCondition(CloudTestRunCreated, metav1.ConditionTrue)

		k6.Status.TestRunID = k6.Spec.TestRunID
	} else {
		k6.UpdateCondition(CloudPLZTestRun, metav1.ConditionFalse)
		// PLZ test run can be defined only via spec.testRunId;
		// otherwise it's not a PLZ test run.
	}
}

func updateCondition(conditions *[]metav1.Condition, conditionType string, conditionStatus metav1.ConditionStatus) {
	reason, ok := reasons[conditionType+string(conditionStatus)]
	if !ok {
		panic(fmt.Sprintf("Invalid condition type and status! `%s` - this should never happen!", conditionType+string(conditionStatus)))
	}
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            "",
	})
}

func (k6 *K6) UpdateCondition(conditionType string, conditionStatus metav1.ConditionStatus) {
	updateCondition(&k6.Status.Conditions, conditionType, conditionStatus)
}

func (k6 K6) IsTrue(conditionType string) bool {
	return meta.IsStatusConditionTrue(k6.Status.Conditions, conditionType)
}

func (k6 K6) IsFalse(conditionType string) bool {
	return meta.IsStatusConditionFalse(k6.Status.Conditions, conditionType)
}

func (k6 K6) IsUnknown(conditionType string) bool {
	return !k6.IsFalse(conditionType) && !k6.IsTrue(conditionType)
}

func (k6 K6) LastUpdate(conditionType string) (time.Time, bool) {
	cond := meta.FindStatusCondition(k6.Status.Conditions, conditionType)
	if cond != nil {
		return cond.LastTransitionTime.Time, true
	}
	return time.Now(), false
}

// SetIfNewer changes k6status only if changes in proposedStatus are consistent
// with the expected progression of a test run. If there were any acceptable
// changes proposed, it returns true.
func (k6status *K6Status) SetIfNewer(proposedStatus K6Status) (isNewer bool) {
	existingConditions := map[string]metav1.Condition{}
	for i := range k6status.Conditions {
		existingConditions[k6status.Conditions[i].Type] = k6status.Conditions[i]
	}

	for _, proposedCondition := range proposedStatus.Conditions {
		// If a new condition is being proposed, just add it to the list.
		if existingCondition, ok := existingConditions[proposedCondition.Type]; !ok {
			k6status.Conditions = append(k6status.Conditions, proposedCondition)
			isNewer = true
		} else {
			// If a change in existing condition is being proposed, check if
			// its timestamp is later than the one in existing condition.
			//
			// Additionally: condition should never return to Unknown status
			// unless it's newly created.

			if proposedCondition.Status != metav1.ConditionUnknown {
				if existingCondition.LastTransitionTime.UnixNano() < proposedCondition.LastTransitionTime.UnixNano() {
					meta.SetStatusCondition(&k6status.Conditions, proposedCondition)
					isNewer = true
				}
			}
		}

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
	}

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
			if proposedStatus.Stage == "started" || proposedStatus.Stage == "finished" || proposedStatus.Stage == "error" {
				k6status.Stage = proposedStatus.Stage
				isNewer = true
			}
		case "started":
			if proposedStatus.Stage == "finished" || proposedStatus.Stage == "error" {
				k6status.Stage = proposedStatus.Stage
				isNewer = true
			}
			// in case of finished or error stage, skip
		}
	}

	return
}
