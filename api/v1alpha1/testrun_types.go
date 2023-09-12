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
	"errors"
	"path/filepath"

	"github.com/grafana/k6-operator/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Stage",type="string",JSONPath=".status.stage",description="Stage"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="TestRunID",type="string",JSONPath=".status.testRunId"

// TestRun is the Schema for the testruns API
type TestRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestRunSpec   `json:"spec,omitempty"`
	Status TestRunStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TestRunList contains a list of TestRun
type TestRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestRun{}, &TestRunList{})
}

// Parse extracts Script data bits from K6 spec and performs basic validation
func (k6 TestRunSpec) ParseScript() (*types.Script, error) {
	spec := k6.Script
	s := &types.Script{
		Filename: "test.js",
		Path:     "/test/",
	}

	if spec.VolumeClaim.Name != "" {
		s.Name = spec.VolumeClaim.Name
		if spec.VolumeClaim.File != "" {
			s.Filename = spec.VolumeClaim.File
		}

		s.Type = "VolumeClaim"
		return s, nil
	}

	if spec.ConfigMap.Name != "" {
		s.Name = spec.ConfigMap.Name

		if spec.ConfigMap.File != "" {
			s.Filename = spec.ConfigMap.File
		}

		s.Type = "ConfigMap"
		return s, nil
	}

	if spec.LocalFile != "" {
		s.Name = "LocalFile"
		s.Type = "LocalFile"
		s.Path, s.Filename = filepath.Split(spec.LocalFile)
		return s, nil
	}

	return nil, errors.New("Script definition should contain one of: ConfigMap, VolumeClaim, LocalFile")
}

// TestRunI implementation for TestRun
func (k6 *TestRun) GetStatus() *TestRunStatus {
	return &k6.Status
}

func (k6 *TestRun) GetSpec() *TestRunSpec {
	return &k6.Spec
}

func (k6 *TestRun) NamespacedName() k8stypes.NamespacedName {
	return k8stypes.NamespacedName{Namespace: k6.Namespace, Name: k6.Name}
}
