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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	// DefaultRunnerImage is the default k6 image used for runner and initializer pods.
	DefaultRunnerImage = "grafana/k6:latest"

	// DefaultStarterImage is the default image used for the starter pod.
	DefaultStarterImage = "ghcr.io/grafana/k6-operator:latest-starter"

	// DefaultParallelism is the default number of runner pods when none is specified.
	DefaultParallelism = int32(1)
)

// TestRunDefaulter implements admission.CustomDefaulter for TestRun.
type TestRunDefaulter struct{}

// +kubebuilder:webhook:path=/mutate-k6-io-v1alpha1-testrun,mutating=true,failurePolicy=fail,sideEffects=None,groups=k6.io,resources=testruns,verbs=create;update,versions=v1alpha1,name=mtestrun.kb.io,admissionReviewVersions=v1

var _ admission.CustomDefaulter = &TestRunDefaulter{}

// SetupTestRunWebhookWithManager registers the defaulting webhook for TestRun.
func SetupTestRunWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&TestRun{}).
		WithDefaulter(&TestRunDefaulter{}).
		Complete()
}

// Default sets sensible defaults on a TestRun when fields are unset.
// This keeps default values in one place rather than scattered across job builders.
func (d *TestRunDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	r, ok := obj.(*TestRun)
	if !ok {
		return fmt.Errorf("expected a TestRun but got %T", obj)
	}

	if r.Spec.Parallelism == 0 {
		r.Spec.Parallelism = DefaultParallelism
	}

	if r.Spec.Runner.Image == "" {
		r.Spec.Runner.Image = DefaultRunnerImage
	}

	if r.Spec.Starter.Image == "" {
		r.Spec.Starter.Image = DefaultStarterImage
	}

	return nil
}
