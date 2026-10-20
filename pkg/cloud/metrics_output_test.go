package cloud

import (
	"encoding/base64"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestTestRunData_Output(t *testing.T) {
	t.Parallel()

	if got := (&TestRunData{}).Output(); got != "cloud" {
		t.Errorf("Output() without metrics output = %q, want %q", got, "cloud")
	}

	trd := TestRunData{MetricsOutput: &MetricsOutput{Endpoint: "otlp.example.com:4318"}}
	if got := trd.Output(); got != "opentelemetry" {
		t.Errorf("Output() with metrics output = %q, want %q", got, "opentelemetry")
	}
}

func TestMetricsOutput_envVars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mo       MetricsOutput
		expected []corev1.EnvVar // order of env vars is important
	}{
		{
			name: "all fields are set",
			mo: MetricsOutput{
				Endpoint: "otlp.example.com:4318",
				URLPath:  "/custom/v1/metrics",
				Insecure: true,
				Username: "user",
				Password: "password",
			},
			expected: []corev1.EnvVar{
				{Name: "K6_OTEL_EXPORTER_PROTOCOL", Value: "http/protobuf"},
				{Name: "K6_OTEL_HTTP_EXPORTER_ENDPOINT", Value: "otlp.example.com:4318"},
				{Name: "K6_OTEL_HTTP_EXPORTER_URL_PATH", Value: "/custom/v1/metrics"},
				{Name: "K6_OTEL_HTTP_EXPORTER_INSECURE", Value: "true"},
				{Name: "K6_OTEL_HEADERS", Value: "Authorization=Basic " + base64.StdEncoding.EncodeToString([]byte("user:password"))},
				{Name: "K6_OTEL_METRIC_PREFIX", Value: "k6_"},
				{Name: "OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION", Value: "base2_exponential_bucket_histogram"},
			},
		},
		{
			name: "only endpoint is set",
			mo:   MetricsOutput{Endpoint: "otlp.example.com"},
			expected: []corev1.EnvVar{
				{Name: "K6_OTEL_EXPORTER_PROTOCOL", Value: "http/protobuf"},
				{Name: "K6_OTEL_HTTP_EXPORTER_ENDPOINT", Value: "otlp.example.com"},
				{Name: "K6_OTEL_METRIC_PREFIX", Value: "k6_"},
				{Name: "OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION", Value: "base2_exponential_bucket_histogram"},
			},
		},
		{
			name: "escape `$` in literals",
			mo: MetricsOutput{
				Endpoint: "host-$(EVIL):4318",
				URLPath:  "/custom/$(EVIL)",
			},
			expected: []corev1.EnvVar{
				{Name: "K6_OTEL_EXPORTER_PROTOCOL", Value: "http/protobuf"},
				{Name: "K6_OTEL_HTTP_EXPORTER_ENDPOINT", Value: "host-$$(EVIL):4318"},
				{Name: "K6_OTEL_HTTP_EXPORTER_URL_PATH", Value: "/custom/$$(EVIL)"},
				{Name: "K6_OTEL_METRIC_PREFIX", Value: "k6_"},
				{Name: "OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION", Value: "base2_exponential_bucket_histogram"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.mo.envVars()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("envVars() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestTestRunData_Preprocess_metricsOutput(t *testing.T) {
	t.Parallel()

	oneLZ := LZDistribution{"label": Distribution{LoadZone: "zone", Percent: 100}}

	t.Run("misconfigured metrics output errors out", func(t *testing.T) {
		t.Parallel()
		for _, mo := range []*MetricsOutput{
			{},
			{Endpoint: "http://otlp.example.com:4318"},
			{Endpoint: "otlp.example.com:4318", Username: "user"},
			{Endpoint: "otlp.example.com:4318", Password: "password"},
		} {
			trd := TestRunData{LZDistribution: oneLZ, MetricsOutput: mo}
			if err := trd.Preprocess(); err == nil {
				t.Errorf("expected error for metrics output %+v", *mo)
			}
		}
	})

	t.Run("fill in args and env vars for k6 process", func(t *testing.T) {
		t.Parallel()
		mo := &MetricsOutput{
			Endpoint: "otlp.example.com:4318",
			URLPath:  "/custom/v1/metrics",
			Insecure: true,
		}
		trd := TestRunData{
			TestRunId:      42,
			LZDistribution: oneLZ,
			MetricsOutput:  mo,
			LZConfig: LZConfig{CLIArgs: CLIArgs{Tags: map[string]string{
				"env":         "staging",
				"test_run_id": "user-value", // reserved: overwritten by the test run ID
			}}},
		}
		if err := trd.Preprocess(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedTagArgs := []string{"--tag", "env=staging", "--tag", "load_zone=zone", "--tag", "test_run_id=42"}
		if !reflect.DeepEqual(trd.TagArgs, expectedTagArgs) {
			t.Errorf("TagArgs = %q, want %q", trd.TagArgs, expectedTagArgs)
		}

		// metrics output env vars come first, and K6_CLOUD_HOST
		// is still set: k6-operator itself relies on it
		expectedEnvVars := append(mo.envVars(),
			corev1.EnvVar{Name: "K6_CLOUD_API_VERSION", Value: "2"},
			corev1.EnvVar{Name: "K6_CLOUD_AGGREGATION_PERIOD", Value: "0s"},
			corev1.EnvVar{Name: "K6_CLOUD_AGGREGATION_WAIT_PERIOD", Value: "0s"},
			corev1.EnvVar{Name: "K6_CLOUD_METRIC_PUSH_INTERVAL", Value: "0s"},
			corev1.EnvVar{Name: "K6_CLOUD_METRIC_PUSH_CONCURRENCY", Value: "0"},
			corev1.EnvVar{Name: "K6_CLOUD_HOST", Value: "https://ingest.k6.io"},
		)
		if !reflect.DeepEqual(trd.RunnerEnvVars, expectedEnvVars) {
			t.Errorf("RunnerEnvVars = %+v, want %+v", trd.RunnerEnvVars, expectedEnvVars)
		}
	})

	t.Run("no metrics output keeps the cloud setup", func(t *testing.T) {
		t.Parallel()
		trd := TestRunData{
			TestRunId:      42,
			LZDistribution: oneLZ,
			LZConfig: LZConfig{CLIArgs: CLIArgs{Tags: map[string]string{
				"test_run_id": "user-value",
			}}},
		}
		if err := trd.Preprocess(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedTagArgs := []string{"--tag", "load_zone=zone", "--tag", "test_run_id=user-value"}
		if !reflect.DeepEqual(trd.TagArgs, expectedTagArgs) {
			t.Errorf("TagArgs = %q, want %q", trd.TagArgs, expectedTagArgs)
		}
		for _, ev := range trd.RunnerEnvVars {
			if strings.HasPrefix(ev.Name, "K6_OTEL_") || strings.HasPrefix(ev.Name, "OTEL_") {
				t.Errorf("unexpected OpenTelemetry env var without metrics output: %+v", ev)
			}
		}
	})
}
