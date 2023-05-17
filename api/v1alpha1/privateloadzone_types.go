/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PrivateLoadZoneSpec defines the desired state of PrivateLoadZone
type PrivateLoadZoneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of PrivateLoadZone. Edit privateloadzone_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// PrivateLoadZoneStatus defines the observed state of PrivateLoadZone
type PrivateLoadZoneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PrivateLoadZone is the Schema for the privateloadzones API
type PrivateLoadZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrivateLoadZoneSpec   `json:"spec,omitempty"`
	Status PrivateLoadZoneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PrivateLoadZoneList contains a list of PrivateLoadZone
type PrivateLoadZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrivateLoadZone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PrivateLoadZone{}, &PrivateLoadZoneList{})
}
