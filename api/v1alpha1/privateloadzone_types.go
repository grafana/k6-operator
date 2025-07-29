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

	guuid "github.com/google/uuid"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/pkg/cloud"

	"go.k6.io/k6/cloudapi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrivateLoadZoneSpec defines the desired state of PrivateLoadZone
type PrivateLoadZoneSpec struct {
	// Grafana Cloud k6 token
	// +kubebuilder:validation:Type=string
	Token string `json:"token"`

	Resources corev1.ResourceRequirements `json:"resources"`

	// Service account name which should be associated with all created Pods.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Node selector which should be applied to all created Pods.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// The Docker image of the k6 runners.
	// Default is "grafana/k6:latest", set by Grafana Cloud k6.
	Image string `json:"image,omitempty"`

	// The imagePullSecrets which should be configured for all created Pods.
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Configuration of the test runs specific for this `PrivateLoadZone`.
	Config PrivateLoadZoneConfig `json:"config,omitempty"`
}

// PrivateLoadZoneStatus defines the observed state of PrivateLoadZone
type PrivateLoadZoneStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:object:generate=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Registered",type="string",JSONPath=".status.conditions[0].status",description="The status of registration"

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

type PrivateLoadZoneConfig struct {
	// Secrets contains a list of definitions copied from `corev1.EnvFromSource`.
	// They are re-packed into `EnvFromSource` struct during TestRun creation.
	Secrets []PLZSecretsConfig `json:"secrets,omitempty"`
}

type PLZSecretsConfig struct {
	ConfigMapRef *corev1.ConfigMapEnvSource `json:"configMapRef,omitempty"`
	SecretRef    *corev1.SecretEnvSource    `json:"secretRef,omitempty"`
}

func init() {
	SchemeBuilder.Register(&PrivateLoadZone{}, &PrivateLoadZoneList{})
}

// Register attempts to register PLZ with the k6 Cloud.
// Regardless of the result, condition PLZRegistered will be set to False.
// Callee is expected to check the returned error and set condition when it's appropriate.
func (plz *PrivateLoadZone) Register(ctx context.Context, logger logr.Logger, client *cloudapi.Client) (string, error) {
	plz.UpdateCondition(PLZRegistered, metav1.ConditionFalse)

	uid := uuid()
	data := cloud.PLZRegistrationData{
		LoadZoneID: plz.Name,
		Resources: cloud.PLZResources{
			CPU:    plz.Spec.Resources.Limits.Cpu().String(),
			Memory: plz.Spec.Resources.Limits.Memory().String(),
		},
		LZConfig: cloud.LZConfig{
			RunnerImage: plz.Spec.Image,
		},
		UID: uid,
	}

	if err := cloud.RegisterPLZ(client, data); err != nil {
		logger.Error(err, fmt.Sprintf("Failed to register PLZ %s.", plz.Name))
		return "", err
	}

	logger.Info(fmt.Sprintf("Registered PLZ %s.", plz.Name))

	return uid, nil
}

// Deregister attempts to deregister PLZ with the k6 Cloud.
// It is meant to be used as a finalizer.
func (plz *PrivateLoadZone) Deregister(ctx context.Context, logger logr.Logger, client *cloudapi.Client) error {
	if err := cloud.DeRegisterPLZ(client, plz.Name); err != nil {
		logger.Error(err, fmt.Sprintf("Failed to de-register PLZ %s.", plz.Name))
		return err
	}

	logger.Info(fmt.Sprintf("De-registered PLZ %s.", plz.Name))

	return nil
}

func uuid() string {
	return guuid.New().String()
}

func (plzConfig *PrivateLoadZoneConfig) ToEnvFromSource() []corev1.EnvFromSource {
	envFromSource := make([]corev1.EnvFromSource, len(plzConfig.Secrets))
	for i := range plzConfig.Secrets {
		envFromSource[i].ConfigMapRef = plzConfig.Secrets[i].ConfigMapRef
		envFromSource[i].SecretRef = plzConfig.Secrets[i].SecretRef
	}
	return envFromSource
}
