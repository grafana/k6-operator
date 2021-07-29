package jobs

import (
	"reflect"
	"testing"
)

func TestNewLabels(t *testing.T) {

	expectedOutcome := map[string]string{
		"app":   "k6",
		"k6_cr": "test",
	}
	labels := newLabels("test")
	if !reflect.DeepEqual(labels, expectedOutcome) {
		t.Errorf("new labels were incorrect, got: %v, want: %v.", labels, expectedOutcome)
	}
}
