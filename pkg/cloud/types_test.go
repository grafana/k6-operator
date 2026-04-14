package cloud

import (
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestInspectOutput_TestNameAndProjectID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		fields            []byte
		expectedProjectID int64
		expectedName      string
	}{
		{
			name:              "empty",
			fields:            []byte(`{}`),
			expectedProjectID: 0,
		},
		{
			name:              "only legacy way of defining the options",
			fields:            []byte(`{"ext":{"loadimpact":{"name":"test","projectID":123}}}`),
			expectedProjectID: 123,
			expectedName:      "test",
		},
		{
			name:              "only new way of defining the options",
			fields:            []byte(`{"cloud":{"name":"lorem","projectID":321}}`),
			expectedProjectID: 321,
			expectedName:      "lorem",
		},
		{
			name:              "both way, priority to new way",
			fields:            []byte(`{"cloud":{"name":"ipsum","projectID":987},"ext":{"loadimpact":{"name":"test","projectID":123}}}`),
			expectedProjectID: 987,
			expectedName:      "ipsum",
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var io *InspectOutput
			if err := json.Unmarshal(tt.fields, &io); err != nil {
				t.Errorf("error unmarshalling json: %v", err)
			}

			if got := io.ProjectID(); got != tt.expectedProjectID {
				t.Errorf("InspectOutput.ProjectID() = %v, want %v", got, tt.expectedProjectID)
			}

			if got := io.TestName(); got != tt.expectedName {
				t.Errorf("InspectOutput.TestName() = %v, want %v", got, tt.expectedName)
			}
		})
	}
}

func TestTestRunData_SecretsEnvVars(t *testing.T) {
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
			got := tt.trd.SecretsEnvVars()
			if len(got) != len(tt.expected) {
				t.Fatalf("SecretsEnvVars() len = %d, want %d; got %+v", len(got), len(tt.expected), got)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("SecretsEnvVars()[%d] = %+v, want %+v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestInspectOutput_SetTestName(t *testing.T) {
	t.Parallel()

	io := &InspectOutput{}
	if got := io.TestName(); got != "" {
		t.Errorf("InspectOutput.TestName() = %v, want empty name", got)
	}

	io.SetTestName("test-lore-ipsum")
	if got := io.TestName(); got != "test-lore-ipsum" {
		t.Errorf("InspectOutput.TestName() = %v, want test-lore-ipsum", got)
	}
}
