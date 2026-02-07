package types

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func UpdateCondition(conditions *[]metav1.Condition, conditionType string, conditionStatus metav1.ConditionStatus) {
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

// SetIfNewer changes cond only if changes in proposedCond are consistent
// with the expected change of conditions both logically and chronologically.
// callbackF can be provided to run a custom function during the loop
// over proposedCond.
// If there were any acceptable changes proposed, it returns true.
func SetIfNewer(cond *[]metav1.Condition,
	proposedCond []metav1.Condition,
	callbackF func(metav1.Condition) bool) (isNewer bool) {

	existingConditions := map[string]metav1.Condition{}
	for i := range *cond {
		existingConditions[(*cond)[i].Type] = (*cond)[i]
	}

	for _, proposedCondition := range proposedCond {
		// If a new condition is being proposed, just add it to the list.
		if existingCondition, ok := existingConditions[proposedCondition.Type]; !ok {
			*cond = append(*cond, proposedCondition)
			isNewer = true
		} else {
			// If a change in existing condition is being proposed, check if
			// its timestamp is later than the one in existing condition.
			//
			// Additionally: condition should never return to Unknown status
			// unless it's newly created.

			if proposedCondition.Status != metav1.ConditionUnknown {
				if existingCondition.LastTransitionTime.UnixNano() < proposedCondition.LastTransitionTime.UnixNano() {
					meta.SetStatusCondition(cond, proposedCondition)
					isNewer = true
				}
			}
		}

		if callbackF != nil {
			if callbackResult := callbackF(proposedCondition); callbackResult {
				isNewer = callbackResult
			}
		}
	}

	return
}

var reasons = map[string]string{
	"TestRunRunningUnknown": "TestRunPreparation",
	"TestRunRunningTrue":    "TestRunRunningTrue",
	"TestRunRunningFalse":   "TestRunRunningFalse",

	"TeardownExecutedUnknown": "TestRunPreparation",
	"TeardownExecutedFalse":   "TeardownExecutedFalse",
	"TeardownExecutedTrue":    "TeardownExecutedTrue",

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

	"CloudTestRunAbortedUnknown": "CloudTestRunAbortedUnknown",
	"CloudTestRunAbortedTrue":    "CloudTestRunAbortedTrue",
	"CloudTestRunAbortedFalse":   "CloudTestRunAbortedFalse",
}
