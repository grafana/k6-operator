package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/grafana/k6-operator/pkg/types"
)

func Test_GetToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		spec     TestRunSpec
		expected string
	}{
		{"only top-level", TestRunSpec{Token: "old"}, "old"},
		{"only cloud section", TestRunSpec{Cloud: &CloudSpec{Token: "new"}}, "new"},
		{"cloud section preferred over top-level", TestRunSpec{Token: "old", Cloud: &CloudSpec{Token: "new"}}, "new"},
		{"empty cloud section", TestRunSpec{Token: "old", Cloud: &CloudSpec{}}, "old"},
		{"no token", TestRunSpec{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.spec.GetToken(); got != tt.expected {
				t.Errorf("GetToken() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func Test_GetTestRunID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		spec     TestRunSpec
		expected string
	}{
		{"only top-level", TestRunSpec{TestRunID: "old"}, "old"},
		{"only cloud section", TestRunSpec{Cloud: &CloudSpec{TestRunID: "new"}}, "new"},
		{"cloud section preferred over top-level", TestRunSpec{TestRunID: "old", Cloud: &CloudSpec{TestRunID: "new"}}, "new"},
		{"cloud empty section", TestRunSpec{TestRunID: "old", Cloud: &CloudSpec{}}, "old"},
		{"no test run id", TestRunSpec{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.spec.GetTestRunID(); got != tt.expected {
				t.Errorf("GetTestRunID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func Test_IsCloudStream(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		spec     TestRunSpec
		expected bool
	}{
		{"no cloud section", TestRunSpec{}, false},
		{"stream false", TestRunSpec{Cloud: &CloudSpec{Stream: false}}, false},
		{"stream true", TestRunSpec{Cloud: &CloudSpec{Stream: true}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.spec.IsCloudStream(); got != tt.expected {
				t.Errorf("IsCloudStream() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func Test_IsCloudTest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		spec     TestRunSpec
		expected bool
	}{
		{"OSS test", TestRunSpec{}, false},
		{"--out cloud only", TestRunSpec{Arguments: "--out cloud"}, true},
		{"cloud.stream only", TestRunSpec{Cloud: &CloudSpec{Stream: true}}, true},
		{"both --out cloud and stream (blocked by early validation in practice)",
			TestRunSpec{Arguments: "--out cloud", Cloud: &CloudSpec{Stream: true}}, true},
		{"cloud without stream", TestRunSpec{Cloud: &CloudSpec{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.spec.IsCloudTest(); got != tt.expected {
				t.Errorf("IsCloudTest() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func Test_Validate_CloudConfigConflicts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    TestRunSpec
		wantErr bool
	}{
		{"OSS test", TestRunSpec{}, false},
		{"matching token", TestRunSpec{Token: "t", Cloud: &CloudSpec{Token: "t"}}, false},
		{"conflicting token", TestRunSpec{Token: "old", Cloud: &CloudSpec{Token: "new"}}, true},
		{"matching test run id", TestRunSpec{TestRunID: "1", Cloud: &CloudSpec{TestRunID: "1"}}, false},
		{"conflicting test run id", TestRunSpec{TestRunID: "1", Cloud: &CloudSpec{TestRunID: "2"}}, true},
		{"--out cloud + stream", TestRunSpec{Arguments: "--out cloud", Cloud: &CloudSpec{Stream: true}}, true},
		{"--out cloud without stream", TestRunSpec{Arguments: "--out cloud"}, false},
		{"stream without --out cloud", TestRunSpec{Cloud: &CloudSpec{Stream: true}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
