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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodMetadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type Pod struct {
	// +optional
	// +kubebuilder:default=false
	Disabled                     bool                              `json:"disabled,omitempty"`
	Affinity                     *corev1.Affinity                  `json:"affinity,omitempty"`
	AutomountServiceAccountToken string                            `json:"automountServiceAccountToken,omitempty"`
	Env                          []corev1.EnvVar                   `json:"env,omitempty"`
	Image                        string                            `json:"image,omitempty"`
	ImagePullSecrets             []corev1.LocalObjectReference     `json:"imagePullSecrets,omitempty"`
	ImagePullPolicy              corev1.PullPolicy                 `json:"imagePullPolicy,omitempty"`
	Metadata                     PodMetadata                       `json:"metadata,omitempty"`
	NodeSelector                 map[string]string                 `json:"nodeSelector,omitempty"`
	Tolerations                  []corev1.Toleration               `json:"tolerations,omitempty"`
	TopologySpreadConstraints    []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	Resources                    corev1.ResourceRequirements       `json:"resources,omitempty"`
	ServiceAccountName           string                            `json:"serviceAccountName,omitempty"`
	SecurityContext              corev1.PodSecurityContext         `json:"securityContext,omitempty"`
	ContainerSecurityContext     corev1.SecurityContext            `json:"containerSecurityContext,omitempty"`
	EnvFrom                      []corev1.EnvFromSource            `json:"envFrom,omitempty"`
	ReadinessProbe               *corev1.Probe                     `json:"readinessProbe,omitempty"`
	LivenessProbe                *corev1.Probe                     `json:"livenessProbe,omitempty"`
	InitContainers               []InitContainer                   `json:"initContainers,omitempty"`
	Volumes                      []corev1.Volume                   `json:"volumes,omitempty"`
	VolumeMounts                 []corev1.VolumeMount              `json:"volumeMounts,omitempty"`
	PriorityClassName            string                            `json:"priorityClassName,omitempty"`
}

type InitContainer struct {
	Name          string                         `json:"name,omitempty"`
	Image         string                         `json:"image,omitempty"`
	Env           []corev1.EnvVar                `json:"env,omitempty"`
	EnvFrom       []corev1.EnvFromSource         `json:"envFrom,omitempty"`
	Command       []string                       `json:"command,omitempty"`
	Args          []string                       `json:"args,omitempty"`
	WorkingDir    string                         `json:"workingDir,omitempty"`
	VolumeMounts  []corev1.VolumeMount           `json:"volumeMounts,omitempty"`
	Resources     corev1.ResourceRequirements    `json:"resources,omitempty"`
	RestartPolicy *corev1.ContainerRestartPolicy `json:"restartPolicy,omitempty"`
}

type K6Scuttle struct {
	Enabled                 string `json:"enabled,omitempty"`
	EnvoyAdminApi           string `json:"envoyAdminApi,omitempty"`
	NeverKillIstio          bool   `json:"neverKillIstio,omitempty"`
	NeverKillIstioOnFailure bool   `json:"neverKillIstioOnFailure,omitempty"`
	DisableLogging          bool   `json:"disableLogging,omitempty"`
	StartWithoutEnvoy       bool   `json:"startWithoutEnvoy,omitempty"`
	WaitForEnvoyTimeout     string `json:"waitForEnvoyTimeout,omitempty"`
	IstioQuitApi            string `json:"istioQuitApi,omitempty"`
	GenericQuitEndpoint     string `json:"genericQuitEndpoint,omitempty"`
	QuitWithoutEnvoyTimeout string `json:"quitWithoutEnvoyTimeout,omitempty"`
}

// CloudSpec defines cloud-related configuration for k6 test runs.
type CloudSpec struct {
	// Token is the name of the Secret that contains the k6 Cloud token.
	Token string `json:"token,omitempty"`

	// TestRunID is the k6 Cloud test run ID. Reserved by Grafana Cloud k6.
	TestRunID string `json:"testRunId,omitempty"`

	// Stream enables streaming results to k6 Cloud via `k6 cloud run --local-execution`.
	Stream bool `json:"stream,omitempty"`
}

// TestRunSpec defines the desired state of TestRun
type TestRunSpec struct {
	// Script describes where the k6 script is located.
	Script K6Script `json:"script"`

	// Parallelism shows the number of k6 runners.
	Parallelism int32 `json:"parallelism"`

	// Separate is a quick way to run all k6 runners on different hostnames
	// using the podAntiAffinity rule.
	Separate bool `json:"separate,omitempty"`

	// Arguments to pass to the k6 process.
	Arguments string `json:"arguments,omitempty"`

	// Port to configure on all k6 containers.
	// Port 6565 is always configured for k6 processes.
	Ports []corev1.ContainerPort `json:"ports,omitempty"`

	// Configuration for the initializer Pod. If omitted, the initializer
	// is configured with the same parameters as a runner Pod.
	Initializer *Pod `json:"initializer,omitempty"`

	// Configuration for the starter Pod.
	Starter Pod `json:"starter,omitempty"`

	// Configuration for a runner Pod.
	Runner Pod `json:"runner,omitempty"`

	// Quiet is a boolean variable that allows to swtich off passing the `--quiet` to k6.
	// +kubebuilder:default="true"
	Quiet string `json:"quiet,omitempty"`

	// Paused is a boolean variable that allows to switch off passing the `--paused` to k6.
	// Use with caution as it can skew the result of the test.
	// +kubebuilder:default="true"
	Paused string `json:"paused,omitempty"`

	// Configuration for Envoy proxy.
	Scuttle K6Scuttle `json:"scuttle,omitempty"`

	Cleanup Cleanup `json:"cleanup,omitempty"`

	// Cloud contains cloud-related configuration. Preferred over the
	// top-level Token and TestRunID fields.
	// +optional
	Cloud *CloudSpec `json:"cloud,omitempty"`

	// Use cloud.testRunId instead.
	// TestRunID is reserved by Grafana Cloud k6. Do not set it manually.
	TestRunID string `json:"testRunId,omitempty"` // PLZ reserved field

	// Use cloud.token instead.
	// Token is reserved by Grafana Cloud k6. Do not set it manually.
	Token string `json:"token,omitempty"` // PLZ reserved field (for now)
}

// K6Script describes where to find the k6 script.
type K6Script struct {
	VolumeClaim K6VolumeClaim `json:"volumeClaim,omitempty"`
	ConfigMap   K6Configmap   `json:"configMap,omitempty"`
	// LocalFile describes the location of the script in the runner image.
	LocalFile string `json:"localFile,omitempty"`
}

// K6VolumeClaim describes the location of the script on the Volume.
type K6VolumeClaim struct {
	// Name of the persistent volumeClaim where the script is stored.
	// It is mounted as a `/test` folder to all k6 Pods.
	Name string `json:"name"`
	// Name of the file to execute (.js or .tar), stored on the Volume.
	// Can include a path component (e.g., "subdir/script.js").
	File string `json:"file,omitempty"`
	// ReadOnly shows whether the volume should be mounted as `readOnly`.
	ReadOnly bool `json:"readOnly,omitempty"`
}

// K6Configmap describes the location of the script in the ConfigMap.
type K6Configmap struct {
	// Name of the ConfigMap. It is expected to be in the sanme namespace as the `TestRun`.
	Name string `json:"name"`
	// Name of the file to execute (.js or .tar), stored as a key in the ConfigMap.
	File string `json:"file,omitempty"`
}

//TODO: cleanup pre-execution?

// Cleanup allows for automatic cleanup of resources post execution.
// +kubebuilder:validation:Enum=post
type Cleanup string

// Stage describes which stage of the test execution lifecycle k6 runners are in.
// +kubebuilder:validation:Enum=initialization;initialized;created;started;stopped;finished;error
type Stage string

// TestRunStatus defines the observed state of TestRun.
type TestRunStatus struct {
	Stage           Stage  `json:"stage,omitempty"`
	TestRunID       string `json:"testRunId,omitempty"`
	AggregationVars string `json:"aggregationVars,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Stage",type="string",JSONPath=".status.stage",description="Stage"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="TestRunID",type="string",JSONPath=".status.testRunId"

// TestRun is the Schema for the testruns API.
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

func (k6 *TestRunSpec) Validate() (*types.CLI, error) {
	cli, err := types.ParseCLI(k6.Arguments)
	if err != nil {
		return nil, err
	}

	if cli.HasCloudOut && k6.IsCloudStream() {
		return nil, errors.New("cannot use both --out cloud and cloud.stream; use cloud.stream instead")
	}

	if k6.Cloud != nil {
		if k6.Cloud.Token != "" && k6.Token != "" && k6.Cloud.Token != k6.Token {
			return nil, errors.New("corrupted TestRun: cloud.token and token are both set with different values")
		}
		if k6.Cloud.TestRunID != "" && k6.TestRunID != "" && k6.Cloud.TestRunID != k6.TestRunID {
			return nil, errors.New("corrupted TestRun: cloud.testRunId and testRunId are both set with different values")
		}
	}

	return cli, nil
}

// Parse extracts Script data bits from K6 spec and performs basic validation
func (k6 TestRunSpec) ParseScript() (*types.Script, error) {
	spec := k6.Script
	s := &types.Script{}

	// VolumeClaim: allow file to include a path component (e.g. "subdir/script.js").
	if spec.VolumeClaim.Name != "" {
		s.Name = spec.VolumeClaim.Name
		if spec.VolumeClaim.File != "" {
			s.Path, s.Filename = filepath.Split(spec.VolumeClaim.File)
			if !filepath.IsAbs(s.Path) {
				// if the path is not absolute, treat this as a default case
				s.Filename = spec.VolumeClaim.File
				s.Path = ""
			}
			s.ReadOnly = spec.VolumeClaim.ReadOnly
		}

		if s.Path == "" {
			s.Path = "/test/"
		}
		if s.Filename == "" {
			s.Filename = "test.js"
		}

		s.Type = "VolumeClaim"
		return s, nil
	}

	// ConfigMap: supports file name only (no path components)
	if spec.ConfigMap.Name != "" {
		s.Path = "/test/"
		s.Filename = "test.js"

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

	return nil, errors.New("script definition should contain one of: ConfigMap, VolumeClaim, LocalFile")
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

// GetTestRunID is a tiny helper to get k6 Cloud test run ID.
// PLZ test run will have test run ID as part of spec,
// while cloud output test run as part of status.
func (k6 *TestRun) GetTestRunID() string {
	specId := k6.GetSpec().GetTestRunID()
	if len(specId) > 0 {
		return specId
	}
	return k6.GetStatus().TestRunID
}

// GetToken returns the cloud token name, preferring cloud.token over the
// deprecated top-level field.
func (spec *TestRunSpec) GetToken() string {
	if spec.Cloud != nil && spec.Cloud.Token != "" {
		return spec.Cloud.Token
	}
	return spec.Token
}

// GetTestRunID returns the cloud test run ID, preferring cloud.testRunId
// over the deprecated top-level field.
func (spec *TestRunSpec) GetTestRunID() string {
	if spec.Cloud != nil && spec.Cloud.TestRunID != "" {
		return spec.Cloud.TestRunID
	}
	return spec.TestRunID
}

// IsCloudStream returns true if cloud streaming is enabled via cloud.stream.
func (spec *TestRunSpec) IsCloudStream() bool {
	return spec.Cloud != nil && spec.Cloud.Stream
}

// IsCloudTest returns true if the test is configured for cloud output,
// either via --out cloud (deprecated) or cloud.stream. Includes PLZ tests.
func (spec *TestRunSpec) IsCloudTest() bool {
	cli, _ := types.ParseCLI(spec.Arguments)
	return cli.HasCloudOut || spec.IsCloudStream()
}

func (k6 *TestRun) ListOptions() *client.ListOptions {
	selector := labels.SelectorFromSet(map[string]string{
		"app":    "k6",
		"k6_cr":  k6.NamespacedName().Name,
		"runner": "true",
	})

	return &client.ListOptions{LabelSelector: selector, Namespace: k6.NamespacedName().Namespace}
}

func (k6 *TestRun) IsInitializerDisabled() bool {
	if k6.GetSpec().Initializer == nil {
		return false
	}
	return k6.GetSpec().Initializer.Disabled
}
