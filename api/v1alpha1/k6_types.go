package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K6Spec defines the desired state of K6
type K6Spec struct {
	Script       string `json:"script"`
	Options      string `json:"options,omitempty"`
	Nodes        int32  `json:"Parallelism"`
}

// K6Status defines the observed state of K6
type K6Status struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// K6 is the Schema for the k6s API
type K6 struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              K6Spec   `json:"spec,omitempty"`
	Status            K6Status `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// K6List contains a list of K6
type K6List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K6 `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K6{}, &K6List{})
}
