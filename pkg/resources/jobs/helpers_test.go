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
			Value: "true",
		},
	}

	envVars := newIstioEnvVar(v1alpha1.K6Scuttle{
		EnvoyAdminApi:       "",
		IstioQuitApi:        "",
		WaitForEnvoyTimeout: "",
		ScuttleLogging:      true,
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
		ScuttleLogging:      false,
	}, true)

	if !reflect.DeepEqual(envVars, expectedOutcome) {
		t.Errorf("new envVars were incorrect, got: %v, want: %v.", envVars, expectedOutcome)
	}
}
