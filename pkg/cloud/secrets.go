package cloud

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// EncodeSecretsConfig encodes the secrets configuration into a pipe-separated
// string for storage in TestRun.Status.SecretsVars.
// Returns an empty string when cfg is nil.
func EncodeSecretsConfig(cfg *SecretsConfig, token string) string {
	if cfg == nil {
		return ""
	}
	return strings.Join([]string{cfg.Endpoint, cfg.ResponsePath, token}, "|")
}

// DecodeSecretsConfig decodes a previously encoded secrets configuration and
// returns the corresponding env vars. Returns nil when encoded is empty.
func DecodeSecretsConfig(encoded string) []corev1.EnvVar {
	if encoded == "" {
		return nil
	}
	parts := strings.SplitN(encoded, "|", 3)
	if len(parts) != 3 {
		return nil
	}
	trData := &TestRunData{
		SecretsToken: parts[2],
		SecretsConfig: &SecretsConfig{
			Endpoint:     parts[0],
			ResponsePath: parts[1],
		},
	}
	return trData.SecretsEnvVars()
}
