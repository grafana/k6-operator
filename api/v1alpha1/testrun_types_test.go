package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/grafana/k6-operator/pkg/types"
)

func Test_ParseScript(t *testing.T) {
	testCases := []struct {
		name        string
		expectedErr bool
		expected    *types.Script
		tr          *TestRunSpec
	}{
		{
			"Empty script",
			true,
			nil,
			&TestRunSpec{},
		},
		{
			"ConfigMap",
			false,
			&types.Script{
				Name:     "Test",
				Path:     "/test/",
				Filename: "thing.js",
				Type:     "ConfigMap",
			},

			&TestRunSpec{
				Script: K6Script{
					ConfigMap: K6Configmap{
						Name: "Test",
						File: "thing.js",
					},
				},
			},
		},
		{
			"LocalFile",
			false,
			&types.Script{
				Name:     "LocalFile",
				Path:     "/custom/",
				Filename: "my_test.js",
				Type:     "LocalFile",
			},

			&TestRunSpec{
				Script: K6Script{
					LocalFile: "/custom/my_test.js",
				},
			},
		},
		{
			"VolumeClaim default case /test/test.js",
			false,
			&types.Script{
				Name:     "Test",
				Path:     "/test/",
				Filename: "thing.js",
				Type:     "VolumeClaim",
			},

			&TestRunSpec{
				Script: K6Script{
					VolumeClaim: K6VolumeClaim{
						Name: "Test",
						File: "thing.js",
					},
				},
			},
		},
		{
			"VolumeClaim custom path /foo/test.js",
			false,
			&types.Script{
				Name:     "Test",
				Path:     "/foo/",
				Filename: "test.js",
				Type:     "VolumeClaim",
			},

			&TestRunSpec{
				Script: K6Script{
					VolumeClaim: K6VolumeClaim{
						Name: "Test",
						File: "/foo/test.js",
					},
				},
			},
		},
		{
			"VolumeClaim default case with subfolders /test/foo/test.js",
			false,
			&types.Script{
				Name:     "Test",
				Path:     "/test/",
				Filename: "foo/test.js",
				Type:     "VolumeClaim",
			},

			&TestRunSpec{
				Script: K6Script{
					VolumeClaim: K6VolumeClaim{
						Name: "Test",
						File: "foo/test.js",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			script, err := testCase.tr.ParseScript()
			if testCase.expectedErr && err == nil {
				t.Errorf("ParseScript should have returned an error.")
			}
			if !testCase.expectedErr && err != nil {
				t.Errorf("ParseScript returned unexpected error: %v", err)
			}
			if !reflect.DeepEqual(script, testCase.expected) {
				t.Errorf("ParseScript failed to return expected output, got: %v, expected: %v", script, testCase.expected)
			}
		})
	}
}
