package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PLZRegistered indicates if the PLZ has been registered.
	// - if empty / Unknown / False, call registration
	// - if True, do nothing
	PLZRegistered = "PLZRegistered"
)

func (plz *PrivateLoadZone) Initialize() {
	t := metav1.Now()
	plz.Status.Conditions = []metav1.Condition{
		metav1.Condition{
			Type:               PLZRegistered,
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: t,
			Reason:             "PLZRegisteredUnknown",
			Message:            "",
		},
	}
}
func (plz PrivateLoadZone) IsTrue(conditionType string) bool {
	return meta.IsStatusConditionTrue(plz.Status.Conditions, conditionType)
}

func (plz PrivateLoadZone) IsFalse(conditionType string) bool {
	return meta.IsStatusConditionFalse(plz.Status.Conditions, conditionType)
}

func (plz PrivateLoadZone) IsUnknown(conditionType string) bool {
	return !plz.IsFalse(conditionType) && !plz.IsTrue(conditionType)
}

func (plz PrivateLoadZone) UpdateCondition(conditionType string, conditionStatus metav1.ConditionStatus) {
	updateCondition(&plz.Status.Conditions, conditionType, conditionStatus)
}

// SetIfNewer changes plzstatus only if changes in proposedStatus are newer.
// If there were any acceptable changes proposed, it returns true.
func (plzstatus *PrivateLoadZoneStatus) SetIfNewer(proposedStatus PrivateLoadZoneStatus) (isNewer bool) {
	existingConditions := map[string]metav1.Condition{}
	for i := range plzstatus.Conditions {
		existingConditions[plzstatus.Conditions[i].Type] = plzstatus.Conditions[i]
	}

	for _, proposedCondition := range proposedStatus.Conditions {
		// If a new condition is being proposed, just add it to the list.
		if existingCondition, ok := existingConditions[proposedCondition.Type]; !ok {
			plzstatus.Conditions = append(plzstatus.Conditions, proposedCondition)
			isNewer = true
		} else {
			// If a change in existing condition is being proposed, check if
			// its timestamp is later than the one in existing condition.
			//
			// Additionally: condition should never return to Unknown status
			// unless it's newly created.

			if proposedCondition.Status != metav1.ConditionUnknown {
				if existingCondition.LastTransitionTime.UnixNano() < proposedCondition.LastTransitionTime.UnixNano() {
					meta.SetStatusCondition(&plzstatus.Conditions, proposedCondition)
					isNewer = true
				}
			}
		}
	}

	return
}
