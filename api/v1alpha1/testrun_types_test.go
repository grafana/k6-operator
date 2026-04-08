package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/grafana/k6-operator/pkg/types"
)

func Test_ParseScript(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		expectedErr bool
		expected    *types.Script
		spec        *TestRunSpec
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
			"ConfigMap with no file defaults to test.js",
			false,
			&types.Script{
				Name:     "Test",
				Path:     "/test/",
				Filename: "test.js",
				Type:     "ConfigMap",
			},
			&TestRunSpec{
				Script: K6Script{
					ConfigMap: K6Configmap{
						Name: "Test",
					},
				},
			},
		},
		{
			"VolumeClaim: default case /test/script.js",
			false,
			&types.Script{
				Name:     "Test",
				Path:     "/test/",
				Filename: "script.js",
				Type:     "VolumeClaim",
			},

			&TestRunSpec{
				Script: K6Script{
					VolumeClaim: K6VolumeClaim{
						Name: "Test",
						File: "script.js",
					},
				},
			},
		},
		{
			"VolumeClaim: custom absolute path",
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
			"VolumeClaim: with relative path",
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
		{
			"VolumeClaim with no file defaults to /test/test.js",
			false,
			&types.Script{
				Name:     "Test",
				Path:     "/test/",
				Filename: "test.js",
				Type:     "VolumeClaim",
			},
			&TestRunSpec{
				Script: K6Script{
					VolumeClaim: K6VolumeClaim{
						Name: "Test",
					},
				},
			},
		},
		{
			"VolumeClaim ReadOnly flag",
			false,
			&types.Script{
				Name:     "Test",
				Path:     "/test/",
				Filename: "script.js",
				Type:     "VolumeClaim",
				ReadOnly: true,
			},
			&TestRunSpec{
				Script: K6Script{
					VolumeClaim: K6VolumeClaim{
						Name:     "Test",
						File:     "script.js",
						ReadOnly: true,
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
			"LocalFile at root path",
			false,
			&types.Script{
				Name:     "LocalFile",
				Path:     "/",
				Filename: "test.js",
				Type:     "LocalFile",
			},
			&TestRunSpec{
				Script: K6Script{
					LocalFile: "/test.js",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			script, err := tt.spec.ParseScript()
			if tt.expectedErr && err == nil {
				t.Errorf("ParseScript should have returned an error.")
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("ParseScript returned unexpected error: %v", err)
			}
			if !reflect.DeepEqual(script, tt.expected) {
				t.Errorf("ParseScript() = %v, want %v", script, tt.expected)
			}
		})
	}
}
