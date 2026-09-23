package controllers

import (
	"testing"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/types"
)

func Test_hasCloudLifecycle(t *testing.T) {
	t.Parallel()

	// newTestRun returns a TestRun with the conditions it has at the start of
	// initialization stage: a PLZ test run is defined by its spec.testRunId.
	newTestRun := func(testRunID string) *v1alpha1.TestRun {
		k6 := &v1alpha1.TestRun{}
		k6.GetSpec().TestRunID = testRunID
		v1alpha1.Initialize(k6)
		return k6
	}

	tests := []struct {
		name     string
		k6       *v1alpha1.TestRun
		cli      *types.CLI
		expected bool
	}{
		{
			name:     "local test run",
			k6:       newTestRun(""),
			cli:      &types.CLI{},
			expected: false,
		},
		{
			name:     "cloud output test run",
			k6:       newTestRun(""),
			cli:      &types.CLI{HasCloudOut: true},
			expected: true,
		},
		{
			name:     "PLZ test run with cloud output",
			k6:       newTestRun("6543"),
			cli:      &types.CLI{HasCloudOut: true},
			expected: true,
		},
		{
			name:     "PLZ test run with OpenTelemetry output",
			k6:       newTestRun("6543"),
			cli:      &types.CLI{},
			expected: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasCloudLifecycle(tt.k6, tt.cli); got != tt.expected {
				t.Errorf("hasCloudLifecycle() = %v, want %v", got, tt.expected)
			}
		})
	}
}
