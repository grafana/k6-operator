package cloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestEncodeSecretsConfig(t *testing.T) {
	t.Parallel()

	endpoint := "https://api.k6.io/provisioning/v1/test_runs/42/decrypt_secret?name={key}"
	respPath := "plaintext"
	token := "abc123"

	tests := []struct {
		name     string
		cfg      *SecretsConfig
		token    string
		expected string
	}{
		{
			name:     "nil cfg returns empty string",
			cfg:      nil,
			token:    token,
			expected: "",
		},
		{
			name:     "valid cfg encodes to pipe-separated string",
			cfg:      &SecretsConfig{Endpoint: endpoint, ResponsePath: respPath},
			token:    token,
			expected: endpoint + "|" + respPath + "|" + token,
		},
		{
			name:     "empty token is preserved in encoding",
			cfg:      &SecretsConfig{Endpoint: endpoint, ResponsePath: respPath},
			token:    "",
			expected: endpoint + "|" + respPath + "|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, EncodeSecretsConfig(tt.cfg, tt.token))
		})
	}
}

func TestDecodeSecretsConfig(t *testing.T) {
	t.Parallel()

	endpoint := "https://api.k6.io/provisioning/v1/test_runs/42/decrypt_secret?name={key}"
	respPath := "plaintext"
	token := "abc123"

	tests := []struct {
		name     string
		encoded  string
		expected []corev1.EnvVar
	}{
		{
			name:     "empty string returns nil",
			encoded:  "",
			expected: nil,
		},
		{
			name:     "malformed string with one part returns nil",
			encoded:  "only-one-part",
			expected: nil,
		},
		{
			name:     "malformed string with two parts returns nil",
			encoded:  "part1|part2",
			expected: nil,
		},
		{
			name:    "valid encoded string returns env vars",
			encoded: endpoint + "|" + respPath + "|" + token,
			expected: []corev1.EnvVar{
				{Name: secretSourceEnvVar, Value: "url"},
				{Name: secretSourceURLTemplate, Value: endpoint},
				{Name: secretSourceURLRespPath, Value: respPath},
				{Name: secretSourceURLAuthKey, Value: "Bearer " + token},
			},
		},
		{
			// SplitN(n=3) means only the first two pipes are used as separators,
			// so a pipe character inside the token is preserved correctly.
			name:    "pipe character in token is preserved by SplitN",
			encoded: endpoint + "|" + respPath + "|token|with|pipes",
			expected: []corev1.EnvVar{
				{Name: secretSourceEnvVar, Value: "url"},
				{Name: secretSourceURLTemplate, Value: endpoint},
				{Name: secretSourceURLRespPath, Value: respPath},
				{Name: secretSourceURLAuthKey, Value: "Bearer token|with|pipes"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, DecodeSecretsConfig(tt.encoded))
		})
	}
}

func TestEncodeDecodeSecretsConfigRoundTrip(t *testing.T) {
	t.Parallel()

	cfg := &SecretsConfig{
		Endpoint:     "https://api.k6.io/provisioning/v1/test_runs/42/decrypt_secret?name={key}",
		ResponsePath: "plaintext",
	}
	token := "abc123"

	encoded := EncodeSecretsConfig(cfg, token)
	got := DecodeSecretsConfig(encoded)

	expected := []corev1.EnvVar{
		{Name: secretSourceEnvVar, Value: "url"},
		{Name: secretSourceURLTemplate, Value: cfg.Endpoint},
		{Name: secretSourceURLRespPath, Value: cfg.ResponsePath},
		{Name: secretSourceURLAuthKey, Value: "Bearer " + token},
	}
	assert.Equal(t, expected, got)
}
