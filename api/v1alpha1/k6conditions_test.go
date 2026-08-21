package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInitializeSetupExecuted(t *testing.T) {
	k6 := &TestRun{}

	Initialize(k6)

	if !IsFalse(k6, SetupExecuted) {
		t.Fatalf("expected %s to be %s", SetupExecuted, metav1.ConditionFalse)
	}

	UpdateCondition(k6, SetupExecuted, metav1.ConditionUnknown)
	if !IsUnknown(k6, SetupExecuted) {
		t.Fatalf("expected %s to be %s", SetupExecuted, metav1.ConditionUnknown)
	}

	UpdateCondition(k6, SetupExecuted, metav1.ConditionTrue)
	if !IsTrue(k6, SetupExecuted) {
		t.Fatalf("expected %s to be %s", SetupExecuted, metav1.ConditionTrue)
	}

	UpdateCondition(k6, SetupExecuted, metav1.ConditionFalse)
	if !IsFalse(k6, SetupExecuted) {
		t.Fatalf("expected %s to be %s", SetupExecuted, metav1.ConditionFalse)
	}
}
