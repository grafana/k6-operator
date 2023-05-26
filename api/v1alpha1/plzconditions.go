package v1alpha1

import (
	"github.com/grafana/k6-operator/pkg/types"
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
	types.UpdateCondition(&plz.Status.Conditions, conditionType, conditionStatus)
}

// SetIfNewer changes plzstatus only if changes in proposedStatus are newer.
// If there were any acceptable changes proposed, it returns true.
func (plzStatus *PrivateLoadZoneStatus) SetIfNewer(proposedStatus PrivateLoadZoneStatus) (isNewer bool) {
	return types.SetIfNewer(&plzStatus.Conditions, proposedStatus.Conditions, nil)
}
