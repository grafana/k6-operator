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

func Test_Argv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		spec     TestRunSpec
		expected []string
	}{
		{
			name:     "nothing set",
			spec:     TestRunSpec{},
			expected: nil,
		},
		{
			name:     "empty args fall back to arguments",
			spec:     TestRunSpec{Arguments: "--vus 10", Args: []string{}},
			expected: []string{"--vus", "10"},
		},
		{
			name:     "args override arguments",
			spec:     TestRunSpec{Arguments: "--vus 10", Args: []string{"--vus", "20"}},
			expected: []string{"--vus", "20"},
		},
		{
			name:     "arguments splitting on spaces",
			spec:     TestRunSpec{Arguments: "--vus 10 --duration 5s"},
			expected: []string{"--vus", "10", "--duration", "5s"},
		},
		{
			name:     "arguments splitting drops empty elements",
			spec:     TestRunSpec{Arguments: "  --vus  10   --duration 5s  "},
			expected: []string{"--vus", "10", "--duration", "5s"},
		},
		{
			name:     "arguments with spaces only",
			spec:     TestRunSpec{Arguments: "   "},
			expected: nil,
		},
		{
			// A YAML block scalar leaves a trailing newline. It must be trimmed.
			name:     "arguments from a block scalar",
			spec:     TestRunSpec{Arguments: "--vus 10 --duration 5s\n"},
			expected: []string{"--vus", "10", "--duration", "5s"},
		},
		{
			name: "args come back verbatim",
			spec: TestRunSpec{Args: []string{
				"--tag", "note=hello world",
				"-e", `QUOTED="value"`,
				"-e", "EMPTY=",
				"--tag", "price=$$100 $(NAME)",
				"",
			}},
			expected: []string{
				"--tag", "note=hello world",
				"-e", `QUOTED="value"`,
				"-e", "EMPTY=",
				"--tag", "price=$$100 $(NAME)",
				"",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			argv := tt.spec.Argv()
			if !reflect.DeepEqual(argv, tt.expected) {
				t.Errorf("Argv() = %#v, want %#v", argv, tt.expected)
			}
		})
	}
}

func Test_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		spec        TestRunSpec
		expectedErr bool
	}{
		{
			name: "exact args",
			spec: TestRunSpec{Args: []string{"--tag", "note=hello world", "-e", "EMPTY="}},
		},
		{
			name:        "empty element in args",
			spec:        TestRunSpec{Args: []string{"--vus", "10", ""}},
			expectedErr: true,
		},
		{
			name:        "only an empty element in args",
			spec:        TestRunSpec{Args: []string{""}},
			expectedErr: true,
		},
		{
			name: "extra spaces in arguments",
			spec: TestRunSpec{Arguments: "  --vus  10   --duration 5s  "},
		},
		{
			name:        "invalid arguments",
			spec:        TestRunSpec{Arguments: "run script.js"},
			expectedErr: true,
		},
		{
			name: "nothing set",
			spec: TestRunSpec{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.spec.Validate()
			if tt.expectedErr && err == nil {
				t.Errorf("Validate() should have returned an error")
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("Validate() returned unexpected error: %v", err)
			}
		})
	}
}
