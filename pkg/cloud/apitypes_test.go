package cloud

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestTestRunData_secretsEnvVars(t *testing.T) {
	t.Parallel()

	someEndpoint := "https://api.k6.io/provisioning/v1/test_runs/42/decrypt_secret?name={key}"
	someRespPath := "plaintext"
	someToken := "abc123"

	tests := []struct {
		name     string
		trd      TestRunData
		expected []corev1.EnvVar // order of env vars is important
	}{
		{
			name:     "nil secrets config returns nil",
			trd:      TestRunData{},
			expected: nil,
		},
		{
			name: "secrets config with token includes auth header",
			trd: TestRunData{
				SecretsConfig: &SecretsConfig{Endpoint: someEndpoint, ResponsePath: someRespPath},
				SecretsToken:  someToken,
			},
			expected: []corev1.EnvVar{
				{Name: secretSourceEnvVar, Value: "url"},
				{Name: secretSourceURLTemplate, Value: someEndpoint},
				{Name: secretSourceURLRespPath, Value: someRespPath},
				{Name: secretSourceURLAuthKey, Value: "Bearer " + someToken},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.trd.secretsEnvVars()
			if len(got) != len(tt.expected) {
				t.Fatalf("secretsEnvVars() len = %d, want %d; got %+v", len(got), len(tt.expected), got)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("secretsEnvVars()[%d] = %+v, want %+v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestLZConfig_reservedEnvVars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		lz       LZConfig
		reserved map[string]struct{}
		expected []corev1.EnvVar
	}{
		{
			name:     "empty GCk6EnvVar",
			lz:       LZConfig{},
			expected: []corev1.EnvVar{},
		},
		{
			name: "no keys match whitelist",
			lz: LZConfig{
				GCk6EnvVars: map[string]string{"NOT_RESERVED": "val"},
			},
			reserved: map[string]struct{}{"RESERVED_A": {}},
			expected: []corev1.EnvVar{},
		},
		{
			name: "all keys match whitelist",
			lz: LZConfig{
				GCk6EnvVars: map[string]string{"VAR_B": "b", "VAR_A": "a"},
			},
			reserved: map[string]struct{}{"VAR_A": {}, "VAR_B": {}},
			expected: []corev1.EnvVar{
				{Name: "VAR_A", Value: "a"},
				{Name: "VAR_B", Value: "b"},
			},
		},
		{
			name: "some keys match whitelist",
			lz: LZConfig{
				GCk6EnvVars: map[string]string{"VAR_B": "b", "SKIP": "no", "VAR_A": "a"},
			},
			reserved: map[string]struct{}{"VAR_B": {}, "VAR_A": {}},
			expected: []corev1.EnvVar{
				{Name: "VAR_A", Value: "a"},
				{Name: "VAR_B", Value: "b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deliberately not parallel: subtests mutate package-level reservedGCk6EnvVars.
			orig := reservedGCk6EnvVars
			reservedGCk6EnvVars = tt.reserved
			defer func() { reservedGCk6EnvVars = orig }()

			got := tt.lz.reservedEnvVars()
			if len(got) != len(tt.expected) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("reservedEnvVars()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestTestRunData_Preprocess(t *testing.T) {
	t.Parallel()

	t.Run("empty or N > 1 load zone error out", func(t *testing.T) {
		t.Parallel()
		trd := TestRunData{}
		if err := trd.Preprocess(); err == nil {
			t.Error("expected error for empty distribution")
		}
		trd.LZDistribution = LZDistribution{
			"a": Distribution{LoadZone: "zone-a", Percent: 50},
			"b": Distribution{LoadZone: "zone-b", Percent: 50},
		}
		if err := trd.Preprocess(); err == nil {
			t.Error("expected error for multiple load zones")
		}
	})

	t.Run("fill in args and env vars for k6 process", func(t *testing.T) {
		t.Parallel()
		trd := TestRunData{
			TestRunId:      42,
			LZDistribution: LZDistribution{"label": Distribution{LoadZone: "zone", Percent: 100}},
			LZConfig: LZConfig{
				Environment: map[string]string{
					"GREETING": "hello world",
				},
				CLIArgs: CLIArgs{
					UserAgent:        "Grafana Cloud k6",
					BlacklistIPs:     []string{"8.8.8.8/32", "1.1.1.1/32"},
					BlockedHostnames: []string{"example.com"},
				},
			},
		}
		if err := trd.Preprocess(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if expected := "--tag load_zone=zone"; trd.TagArgs != expected {
			t.Errorf("TagArgs = %q, want %q", trd.TagArgs, expected)
		}

		// first are org env vars from API, then the reserved env vars
		expectedEnvArgs := `-e GREETING="${K6_CLOUD_OPERATOR_ENV_0}"` +
			` -e K6_CLOUDRUN_DISTRIBUTION="${K6_CLOUD_OPERATOR_ENV_1}"` +
			` -e K6_CLOUDRUN_LOAD_ZONE="${K6_CLOUD_OPERATOR_ENV_2}"` +
			` -e K6_CLOUDRUN_TEST_RUN_ID="${K6_CLOUD_OPERATOR_ENV_3}"`
		if trd.EnvArgs != expectedEnvArgs {
			t.Errorf("EnvArgs = %q, want %q", trd.EnvArgs, expectedEnvArgs)
		}

		expectedEnvVars := []corev1.EnvVar{
			{Name: "K6_USER_AGENT", Value: "Grafana Cloud k6"},
			{Name: "K6_BLACKLIST_IPS", Value: "8.8.8.8/32,1.1.1.1/32"},
			{Name: "K6_BLOCK_HOSTNAMES", Value: "example.com"},
			{Name: "K6_CLOUD_API_VERSION", Value: "2"},
			{Name: "K6_CLOUD_AGGREGATION_PERIOD", Value: "0s"},
			{Name: "K6_CLOUD_AGGREGATION_WAIT_PERIOD", Value: "0s"},
			{Name: "K6_CLOUD_METRIC_PUSH_INTERVAL", Value: "0s"},
			{Name: "K6_CLOUD_METRIC_PUSH_CONCURRENCY", Value: "0"},
			{Name: "K6_CLOUD_HOST", Value: "https://ingest.k6.io"},
			{Name: "K6_CLOUD_OPERATOR_ENV_0", Value: "hello world"},
			{Name: "K6_CLOUD_OPERATOR_ENV_1", Value: "label"},
			{Name: "K6_CLOUD_OPERATOR_ENV_2", Value: "zone"},
			{Name: "K6_CLOUD_OPERATOR_ENV_3", Value: "42"},
		}

		if len(trd.RunnerEnvVars) != len(expectedEnvVars) {
			t.Fatalf("RunnerEnvVars len = %d, want %d: %+v", len(trd.RunnerEnvVars), len(expectedEnvVars), trd.RunnerEnvVars)
		}
		for i := range expectedEnvVars {
			if trd.RunnerEnvVars[i] != expectedEnvVars[i] {
				t.Errorf("RunnerEnvVars[%d] = %v, want %v", i, trd.RunnerEnvVars[i], expectedEnvVars[i])
			}
		}
	})
}
