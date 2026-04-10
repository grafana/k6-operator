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

func TestLZConfig_EnvVars(t *testing.T) {
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

			got := tt.lz.EnvVars()
			if len(got) != len(tt.expected) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("EnvVars()[%d] = %v, want %v", i, got[i], tt.expected[i])
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
