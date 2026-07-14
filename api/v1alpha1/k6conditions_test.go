package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Initialize_PLZDetection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		spec               TestRunSpec
		expectPLZCondition bool
		expectCloudID      string
	}{
		{
			// backwards-compatibility case
			"top-level test run id",
			TestRunSpec{TestRunID: "123"},
			true, "123",
		},
		{
			// current standard approach
			"test run id in cloud section",
			TestRunSpec{Cloud: &CloudSpec{TestRunID: "456"}},
			true, "456",
		},
		{
			// this is tracked by validation as invalid
			"cloud section preferred over top-level",
			TestRunSpec{TestRunID: "111", Cloud: &CloudSpec{TestRunID: "888"}},
			true, "888",
		},
		{
			"no test run id",
			TestRunSpec{},
			false, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k6 := &TestRun{Spec: tt.spec}
			Initialize(k6)

			// check condition for PLZ
			isPLZ := IsTrue(k6, CloudPLZTestRun)
			if isPLZ != tt.expectPLZCondition {
				t.Errorf("PLZ condition = %v, want %v", isPLZ, tt.expectPLZCondition)
			}

			if tt.expectPLZCondition {
				// check invariants for any PLZ test
				if !IsTrue(k6, CloudTestRunCreated) {
					t.Error("expected CloudTestRunCreated=True for PLZ")
				}
				if k6.GetStatus().TestRunID != tt.expectCloudID {
					t.Errorf("status.testRunId = %q, want %q", k6.GetStatus().TestRunID, tt.expectCloudID)
				}
			}
		})
	}
}

func Test_Initialize_AlwaysSetsBaseConditions(t *testing.T) {
	t.Parallel()
	k6 := &TestRun{Spec: TestRunSpec{}}
	Initialize(k6)

	if IsTrue(k6, CloudTestRun) || IsFalse(k6, CloudTestRun) {
		t.Error("expected CloudTestRun=Unknown after Initialize")
	}
	if !IsFalse(k6, CloudTestRunAborted) {
		t.Error("expected CloudTestRunAborted=False after Initialize")
	}
	if !IsFalse(k6, TeardownExecuted) {
		t.Error("expected TeardownExecuted=False after Initialize")
	}

	conds := k6.GetStatus().Conditions
	found := false
	for _, c := range conds {
		if c.Type == TestRunRunning && c.Status == metav1.ConditionUnknown {
			found = true
		}
	}
	if !found {
		t.Error("expected TestRunRunning=Unknown after Initialize")
	}
}
