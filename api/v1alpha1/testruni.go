package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestRunI is meant as abstraction over both TestRun and K6 while
// both types are supported. Consider removing it, when K6 is deprecated.
// +kubebuilder:object:generate=false

type TestRunI interface {
	runtime.Object
	metav1.Object
	client.Object

	GetStatus() *TestRunStatus
	GetSpec() *TestRunSpec
	NamespacedName() types.NamespacedName
	IsPaused() bool
}

// TestRunID is a tiny helper to get k6 Cloud test run ID.
// PLZ test run will have test run ID as part of spec
// while cloud output test run as part of status.
func TestRunID(k6 TestRunI) string {
	specId := k6.GetSpec().TestRunID
	if len(specId) > 0 {
		return specId
	}
	return k6.GetStatus().TestRunID
}

func ListOptions(k6 TestRunI) *client.ListOptions {
	selector := labels.SelectorFromSet(map[string]string{
		"app":    "k6",
		"k6_cr":  k6.NamespacedName().Name,
		"runner": "true",
	})

	return &client.ListOptions{LabelSelector: selector, Namespace: k6.NamespacedName().Namespace}
}
