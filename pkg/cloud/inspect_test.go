package cloud

import (
	"encoding/json"
	"testing"
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
