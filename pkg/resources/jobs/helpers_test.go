package jobs

import (
	"reflect"
	"testing"

	"github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestNewLabels(t *testing.T) {

	expectedOutcome := map[string]string{
		"app":   "k6",
		"k6_cr": "test",
	}
	labels := newLabels("test")
	if !reflect.DeepEqual(labels, expectedOutcome) {
		t.Errorf("new labels were incorrect, got: %v, want: %v.", labels, expectedOutcome)
	}
}

func TestNewIstioCommandIfTrue(t *testing.T) {
	expectedOutcome := []string{"scuttle", "k6", "run"}
	command, _ := newIstioCommand("true", []string{"k6", "run"})

	if diff := deep.Equal(expectedOutcome, command); diff != nil {
		t.Errorf("newIstioCommand returned unexpected data, diff: %s", diff)
	}
}

func TestNewIstioCommandIfFalse(t *testing.T) {
	expectedOutcome := []string{"k6", "run"}
	command, _ := newIstioCommand("false", []string{"k6", "run"})

	if diff := deep.Equal(expectedOutcome, command); diff != nil {
		t.Errorf("newIstioCommand returned unexpected data, diff: %s", diff)
	}
}

func TestNewIstioDefaultEnvVar(t *testing.T) {
	expectedOutcome := []corev1.EnvVar{
		{
			Name:  "ENVOY_ADMIN_API",
			Value: "http://127.0.0.1:15000",
		},
		{
			Name:  "ISTIO_QUIT_API",
			Value: "http://127.0.0.1:15020",
		},
		{
			Name:  "WAIT_FOR_ENVOY_TIMEOUT",
			Value: "15",
		},
	}

	envVars := newIstioEnvVar(v1alpha1.K6Scuttle{
		EnvoyAdminApi:       "",
		IstioQuitApi:        "",
		WaitForEnvoyTimeout: "",
	}, true)

	if !reflect.DeepEqual(envVars, expectedOutcome) {
		t.Errorf("new envVars were incorrect, got: %v, want: %v.", envVars, expectedOutcome)
	}
}

func TestNewIstioEnvVarVaryingTheDefault(t *testing.T) {

	expectedOutcome := []corev1.EnvVar{
		{
			Name:  "ENVOY_ADMIN_API",
			Value: "http://localhost:15020",
		},
		{
			Name:  "ISTIO_QUIT_API",
			Value: "http://127.17.0.1:15020",
		},
		{
			Name:  "WAIT_FOR_ENVOY_TIMEOUT",
			Value: "50",
		},
	}

	envVars := newIstioEnvVar(v1alpha1.K6Scuttle{
		EnvoyAdminApi:       "http://localhost:15020",
		IstioQuitApi:        "http://127.17.0.1:15020",
		WaitForEnvoyTimeout: "50",
	}, true)

	if !reflect.DeepEqual(envVars, expectedOutcome) {
		t.Errorf("new envVars were incorrect, got: %v, want: %v.", envVars, expectedOutcome)
	}
}

func TestNewIstioEnvVarTrueValues(t *testing.T) {
	expectedOutcome := []corev1.EnvVar{
		{
			Name:  "ENVOY_ADMIN_API",
			Value: "http://127.0.0.1:15000",
		},
		{
			Name:  "ISTIO_QUIT_API",
			Value: "http://127.0.0.1:15020",
		},
		{
			Name:  "WAIT_FOR_ENVOY_TIMEOUT",
			Value: "15",
		},
		{
			Name:  "SCUTTLE_LOGGING",
			Value: "false",
		},
	}

	envVars := newIstioEnvVar(v1alpha1.K6Scuttle{
		EnvoyAdminApi:       "",
		IstioQuitApi:        "",
		WaitForEnvoyTimeout: "",
		DisableLogging:      true,
	}, true)

	if !reflect.DeepEqual(envVars, expectedOutcome) {
		t.Errorf("new envVars were incorrect, got: %v, want: %v.", envVars, expectedOutcome)
	}
}

func TestNewIstioEnvVarFalseValues(t *testing.T) {
	expectedOutcome := []corev1.EnvVar{
		{
			Name:  "ENVOY_ADMIN_API",
			Value: "http://127.0.0.1:15000",
		},
		{
			Name:  "ISTIO_QUIT_API",
			Value: "http://127.0.0.1:15020",
		},
		{
			Name:  "WAIT_FOR_ENVOY_TIMEOUT",
			Value: "15",
		},
	}

	envVars := newIstioEnvVar(v1alpha1.K6Scuttle{
		EnvoyAdminApi:       "",
		IstioQuitApi:        "",
		WaitForEnvoyTimeout: "",
		DisableLogging:      false,
	}, true)

	if !reflect.DeepEqual(envVars, expectedOutcome) {
		t.Errorf("new envVars were incorrect, got: %v, want: %v.", envVars, expectedOutcome)
	}
}
func TestConvertEnvVars(t *testing.T) {
	testCases := []struct {
		name     string
		input    []corev1.EnvVar
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []corev1.EnvVar{},
			expected: []string{},
		},
		{
			name: "single env var",
			input: []corev1.EnvVar{
				{Name: "TEST_VAR", Value: "test_value"},
			},
			expected: []string{"TEST_VAR=test_value"},
		},
		{
			name: "multiple env vars",
			input: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
				{Name: "my-env", Value: "myValue"},
			},
			expected: []string{"VAR1=value1", "VAR2=value2", "my-env=myValue"},
		},
		{
			name: "empty value should be skipped",
			input: []corev1.EnvVar{
				{Name: "EMPTY_VAR", Value: ""},
				{Name: "VALID_VAR", Value: "valid_value"},
			},
			expected: []string{"VALID_VAR=valid_value"},
		},
		{
			name: "special characters in value",
			input: []corev1.EnvVar{
				{Name: "SPECIAL", Value: "value with spaces!@#"},
				{Name: "URL", Value: "https://example.com?param=value"},
			},
			expected: []string{"SPECIAL=value with spaces!@#", "URL=https://example.com?param=value"},
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertEnvVarsToStringSlice(tc.input)

			// Check if both are empty (handles nil vs empty slice issue)
			if len(result) == 0 && len(tc.expected) == 0 {
				return // Both empty, test passes
			}

			// For non-empty cases, use reflect.DeepEqual
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("%s failed. Got %v, expected %v", tc.name, result, tc.expected)
			}
		})
	}
}
