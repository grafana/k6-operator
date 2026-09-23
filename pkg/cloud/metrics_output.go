package cloud

import (
	"encoding/base64"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// Settings of the OpenTelemetry output of PLZ runners, where k6 defaults don't fit.
// Everything else is left to k6 defaults.
const (
	otelOutput = "opentelemetry"
	// k6 defaults to gRPC, but many OTLP receivers, hosted ones included, accept only HTTP.
	otelExporterProtocol     = "http/protobuf"
	otelMetricPrefix         = "k6_"
	otelHistogramAggregation = "base2_exponential_bucket_histogram"

	// testRunIDTag scopes the metrics of a test run when they are sent
	// with the OpenTelemetry output, i.e. not via the cloud output.
	testRunIDTag = "test_run_id"
)

// MetricsOutput holds the configuration of the OTLP/HTTP receiver that
// PLZ runners send metrics to with the OpenTelemetry output of k6.
type MetricsOutput struct {
	// Endpoint is the host and optional port of the receiver, without a scheme,
	// e.g. `otlp.example.com:4318`.
	Endpoint string `json:"endpoint"`
	// URLPath is the path of the metrics receiver. If empty, the default of k6 is used.
	URLPath string `json:"url_path,omitempty"`
	// Insecure makes the runners connect to the receiver without TLS.
	Insecure bool `json:"insecure,omitempty"`
	// Username and Password, if set, authenticate the runners to the receiver with
	// basic auth. Like the test run token, they reach the runners as a plain env var.
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// validate checks that the configuration can be passed to k6 as is.
func (mo *MetricsOutput) validate() error {
	if len(mo.Endpoint) == 0 {
		return fmt.Errorf("metrics output must have an endpoint")
	}
	// k6 expects host[:port] here and would fail to connect otherwise.
	if strings.Contains(mo.Endpoint, "://") {
		return fmt.Errorf("metrics output endpoint must be host[:port] without a scheme, provided: %s", mo.Endpoint)
	}
	if (len(mo.Username) == 0) != (len(mo.Password) == 0) {
		return fmt.Errorf("metrics output must have both username and password, or neither")
	}
	return nil
}

// envVars returns the env vars that configure the OpenTelemetry output of k6.
func (mo *MetricsOutput) envVars() []corev1.EnvVar {
	ev := []corev1.EnvVar{
		{Name: "K6_OTEL_EXPORTER_PROTOCOL", Value: otelExporterProtocol},
		{Name: "K6_OTEL_HTTP_EXPORTER_ENDPOINT", Value: escapeKubeExpansion(mo.Endpoint)},
	}
	if len(mo.URLPath) > 0 {
		ev = append(ev, corev1.EnvVar{
			Name: "K6_OTEL_HTTP_EXPORTER_URL_PATH", Value: escapeKubeExpansion(mo.URLPath),
		})
	}
	if mo.Insecure {
		ev = append(ev, corev1.EnvVar{Name: "K6_OTEL_HTTP_EXPORTER_INSECURE", Value: "true"})
	}
	if len(mo.Username) > 0 {
		// The OpenTelemetry output of k6 v1 can authenticate only with headers.
		// Base64 has neither `,` nor `$`, so the value is safe for k6 and Kubernetes.
		credentials := base64.StdEncoding.EncodeToString([]byte(mo.Username + ":" + mo.Password))
		ev = append(ev, corev1.EnvVar{Name: "K6_OTEL_HEADERS", Value: "Authorization=Basic " + credentials})
	}
	ev = append(ev, corev1.EnvVar{Name: "K6_OTEL_METRIC_PREFIX", Value: otelMetricPrefix})
	// This one is read by OpenTelemetry SDK itself.
	ev = append(ev, corev1.EnvVar{
		Name: "OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION", Value: otelHistogramAggregation,
	})
	return ev
}

// Output returns the value of the k6 `--out` option for this test run.
func (trd *TestRunData) Output() string {
	if trd.MetricsOutput != nil {
		return otelOutput
	}
	return "cloud"
}

// preprocessMetricsOutput validates the metrics output of the test run, if any,
// reserves its test_run_id tag and adds the env vars of the OpenTelemetry output.
// It must be called before preprocessTags, which turns the tags into arguments.
func (trd *TestRunData) preprocessMetricsOutput() error {
	if trd.MetricsOutput == nil {
		return nil
	}
	if err := trd.MetricsOutput.validate(); err != nil {
		return err
	}

	if trd.Tags == nil {
		trd.Tags = make(map[string]string)
	}
	// Metrics of all instances of the test run are stored together,
	// so this tag is reserved, like load_zone.
	trd.Tags[testRunIDTag] = trd.TestRunID()

	trd.RunnerEnvVars = append(trd.RunnerEnvVars, trd.MetricsOutput.envVars()...)
	return nil
}
