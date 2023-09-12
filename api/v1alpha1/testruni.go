package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
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
}
