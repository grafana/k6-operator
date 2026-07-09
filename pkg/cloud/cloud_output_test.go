package cloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/cloudapi"
)

func newTestClient(t *testing.T, host string) *cloudapi.Client {
	t.Helper()
	l := logrus.New()
	l.SetOutput(testWriter{t})
	return cloudapi.NewClient(l, "test-token", host, "1.2.3", time.Minute)
}

// testWriter routes logrus output to t.Log so it only appears on failure.
type testWriter struct{ t *testing.T }

func (tw testWriter) Write(p []byte) (int, error) {
	tw.t.Log(string(p))
	return len(p), nil
}

func TestCreateTestRun_SecretsFields(t *testing.T) {
	t.Parallel()

	endpoint := "https://api.k6.io/provisioning/v1/test_runs/42/decrypt_secret?name={key}"
	respPath := "plaintext"
	token := "run-scoped-token"

	tests := []struct {
		name           string
		serverResp     map[string]any
		wantSecretsNil bool
		wantToken      string
		wantEndpoint   string
		wantRespPath   string
	}{
		{
			name: "secrets_config and test_run_token are parsed from response",
			serverResp: map[string]any{
				"reference_id":   "42",
				"config":         map[string]any{},
				"test_run_token": token,
				"secrets_config": map[string]any{
					"endpoint":      endpoint,
					"response_path": respPath,
				},
			},
			wantSecretsNil: false,
			wantToken:      token,
			wantEndpoint:   endpoint,
			wantRespPath:   respPath,
		},
		{
			name: "missing secrets fields are nil/empty (backwards compatibility)",
			serverResp: map[string]any{
				"reference_id": "42",
				"config":       map[string]any{},
			},
			wantSecretsNil: true,
			wantToken:      "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.serverResp)
			}))
			defer srv.Close()

			c := newTestClient(t, srv.URL)
			result, err := createTestRun(c, srv.URL, &TestRun{
				Name:     "test",
				VUsMax:   10,
				Duration: 60,
			})

			require.NoError(t, err)
			assert.Equal(t, "42", result.ReferenceID)
			assert.Equal(t, tt.wantToken, result.SecretsToken)

			if tt.wantSecretsNil {
				assert.Nil(t, result.SecretsConfig)
			} else {
				require.NotNil(t, result.SecretsConfig)
				assert.Equal(t, tt.wantEndpoint, result.SecretsConfig.Endpoint)
				assert.Equal(t, tt.wantRespPath, result.SecretsConfig.ResponsePath)
			}
		})
	}
}

func TestCreateTestRun_MissingReferenceID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"config": map[string]any{}})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := createTestRun(c, srv.URL, &TestRun{Name: "test", VUsMax: 1, Duration: 10})
	assert.EqualError(t, err, "failed to get a reference ID")
}
