package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_SetIfNewer(t *testing.T) {
	var (
		t1 = metav1.Now()
		t2 = metav1.Time{
			Time: t1.Add(time.Nanosecond),
		}
	)

	testCases := []struct {
		name                         string
		oldConditions, newConditions *[]metav1.Condition
		expected                     bool
		expectedConditions           []metav1.Condition
	}{
		{
			"empty case should be negative",
			&[]metav1.Condition{},
			&[]metav1.Condition{},
			false,
			[]metav1.Condition{},
		},
		{
			"setting any new condition should be successful",
			&[]metav1.Condition{},
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t1,
				},
			},
			true,
			[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t1,
				},
			},
		},
		{
			"increasing timestamp without change of condition should be negative",
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t1,
				},
			},
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t2,
				},
			},
			false,
			[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t1,
				},
			},
		},
		{
			"changing condition Unknown -> False should be successul if timestamp increased",
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t1,
				},
			},
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: t2,
				},
			},
			true,
			[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: t2,
				},
			},
		},
		{
			"changing condition Unknown -> False should be negative if timestamp didn't change",
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t1,
				},
			},
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: t1,
				},
			},
			false,
			[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t1,
				},
			},
		},
		{
			"changing condition Unknown -> True should be successul is timestamp increased",
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t1,
				},
			},
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: t2,
				},
			},
			true,
			[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: t2,
				},
			},
		},
		{
			"changing condition True -> False should be successul if timestamp increased",
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: t1,
				},
			},
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: t2,
				},
			},
			true,
			[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: t2,
				},
			},
		},
		{
			"changing condition True -> Unknown should be negative even if timestamp increased",
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: t1,
				},
			},
			&[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: t2,
				},
			},
			false,
			[]metav1.Condition{
				metav1.Condition{
					Type:               "cond",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: t1,
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			isNewer := SetIfNewer(testCase.oldConditions, *testCase.newConditions, nil)
			assert.Equal(t, testCase.expected, isNewer, "testCase", testCase)
			assert.ElementsMatch(t, *testCase.oldConditions, testCase.expectedConditions, "testCase", testCase)
		})

	}
}

// TODO add test for callback in SetIfNewer
