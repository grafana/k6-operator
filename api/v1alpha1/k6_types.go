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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodMetadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// K6Spec defines the desired state of K6
type K6Spec struct {
	Script      K6Script               `json:"script"`
	Parallelism int32                  `json:"parallelism"`
	Separate    bool                   `json:"separate,omitempty"`
	Arguments   string                 `json:"arguments,omitempty"`
	Image       string                 `json:"image,omitempty"`
	Ports       []corev1.ContainerPort `json:"ports,omitempty"`
	Starter     PodMetadata            `json:"starter,omitempty"`
	Runner      PodMetadata            `json:"runner,omitempty"`
	Env         []corev1.EnvVar        `json:"env,omitempty"`
	//	Cleanup     Cleanup `json:"cleanup,omitempty"` // TODO
}

// K6Script describes where the script to execute the tests is found
type K6Script struct {
	VolumeClaim K6VolumeClaim `json:"volumeClaim,omitempty"`
	ConfigMap   K6Configmap   `json:"configMap,omitempty"`
}

// K6VolumeClaim describes the volume claim script location
type K6VolumeClaim struct {
	Name string `json:"name"`
	File string `json:"file,omitempty"`
}

// K6Configmap describes the config map script location
type K6Configmap struct {
	Name string `json:"name"`
	File string `json:"file,omitempty"`
}

// Cleanup allows for automatic cleanup of resources pre or post execution
// +kubebuilder:validation:Enum=pre;post
// type Cleanup string

// Stage describes which stage of the test execution lifecycle our runners are in
// +kubebuilder:validation:Enum=created;started
type Stage string

// K6Status defines the observed state of K6
type K6Status struct {
	Stage Stage `json:"stage,omitempty"`
}

// K6 is the Schema for the k6s API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type K6 struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K6Spec   `json:"spec,omitempty"`
	Status K6Status `json:"status,omitempty"`
}

// K6List contains a list of K6
// +kubebuilder:object:root=true
type K6List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K6 `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K6{}, &K6List{})
}
