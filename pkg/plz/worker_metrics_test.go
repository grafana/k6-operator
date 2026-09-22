package plz

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_plzk6Args_metricsOutput(t *testing.T) {
	t.Parallel()

	trData := &cloud.TestRunData{
		MetricsOutput: &cloud.MetricsOutput{Endpoint: "otlp.example.com:4318"},
		TagArgs:       []string{"--tag", "load_zone=private-zone", "--tag", "test_run_id=1234"},
		LZConfig: cloud.LZConfig{
			CLIArgs: cloud.CLIArgs{IncludeSystemEnvVars: true},
		},
	}
	expected := []string{
		"--out",
		"opentelemetry",
		"--tag",
		"load_zone=private-zone",
		"--tag",
		"test_run_id=1234",
		"--no-thresholds",
		"--log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=private-zone,label.test_run_id=1234,header.Authorization=Token $(K6_CLOUD_TOKEN)",
		"--include-system-env-vars=true",
	}

	if diff := deep.Equal(expected, plzk6Args("private-zone", "1234", trData)); diff != nil {
		t.Error(diff)
	}
}

func Test_complete_metricsOutput(t *testing.T) {
	t.Parallel()

	c, _ := client.New(nil, client.Options{})
	worker := NewPLZWorker(&v1alpha1.PrivateLoadZone{}, "token", c, logr.Logger{})
	tr := worker.template.Create()

	trData := &cloud.TestRunData{
		TestRunId: 6543,
		LZConfig: cloud.LZConfig{
			RunnerImage:   "grafana/k6:1.6.1",
			InstanceCount: 3,
			ArchiveURL:    "https://foo.s3.amazonaws.com",
		},
		LZDistribution: cloud.LZDistribution{
			"some-label": cloud.Distribution{LoadZone: "some-zone", Percent: 100},
		},
		MetricsOutput: &cloud.MetricsOutput{
			Endpoint: "otlp.example.com:4318",
			URLPath:  "/custom/v1/metrics",
			Username: "123456",
			Password: "glc_example-token",
		},
	}
	if err := trData.Preprocess(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	worker.complete(tr, trData)

	if diff := deep.Equal(tr.Spec.Args, plzk6Args("", "6543", trData)); diff != nil {
		t.Errorf("unexpected args, diff: %s", diff)
	}
	if tr.Spec.Args[0] != "--out" || tr.Spec.Args[1] != "opentelemetry" {
		t.Errorf("runners must use the OpenTelemetry output, args: %q", tr.Spec.Args)
	}

	env := make(map[string]string, len(tr.Spec.Runner.Env))
	for _, ev := range tr.Spec.Runner.Env {
		env[ev.Name] = ev.Value
	}
	for name, value := range map[string]string{
		"K6_OTEL_EXPORTER_PROTOCOL":                                "http/protobuf",
		"K6_OTEL_HTTP_EXPORTER_ENDPOINT":                           "otlp.example.com:4318",
		"K6_OTEL_HTTP_EXPORTER_URL_PATH":                           "/custom/v1/metrics",
		"K6_OTEL_HEADERS":                                          "Authorization=Basic MTIzNDU2OmdsY19leGFtcGxlLXRva2Vu",
		"K6_OTEL_METRIC_PREFIX":                                    "k6_",
		"OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION": "base2_exponential_bucket_histogram",
	} {
		if env[name] != value {
			t.Errorf("runner env %s = %q, want %q", name, env[name], value)
		}
	}
	if _, ok := env["K6_OTEL_HTTP_EXPORTER_INSECURE"]; ok {
		t.Errorf("runner env must not disable TLS unless metrics output is insecure")
	}
}
