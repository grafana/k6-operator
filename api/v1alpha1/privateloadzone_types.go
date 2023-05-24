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
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"go.k6.io/k6/cloudapi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrivateLoadZoneSpec defines the desired state of PrivateLoadZone
type PrivateLoadZoneSpec struct {
	Token              string              `json:"token"`
	Resources          corev1.ResourceList `json:"resources"`
	ServiceAccountName string              `json:"serviceAccountName,omitempty"`
	NodeSelector       map[string]string   `json:"nodeSelector,omitempty"`
}

// PrivateLoadZoneStatus defines the observed state of PrivateLoadZone
type PrivateLoadZoneStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
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

// Register attempts to register PLZ with the k6 Cloud.
// Regardless of the result, condition PLZRegistered will be updated.
func (plz *PrivateLoadZone) Register(ctx context.Context, logger logr.Logger, client *cloudapi.Client) {
	plz.UpdateCondition(PLZRegistered, metav1.ConditionFalse)

	// TODO add register call and error processing

	plz.UpdateCondition(PLZRegistered, metav1.ConditionTrue)
}

// Deregister attempts to deregister PLZ with the k6 Cloud.
// It is meant to be used as a finalizer.
func (plz *PrivateLoadZone) Deregister(ctx context.Context, logger logr.Logger, client *cloudapi.Client) {
	// TODO add deregister call and error processing

	fmt.Println("calling deregister for", *plz)
	plz.UpdateCondition(PLZRegistered, metav1.ConditionFalse)
}
