package types

import (
	"testing"

	deep "github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_Script_Volume(t *testing.T) {
	tests := []struct {
		name     string
		script   Script
		expected []corev1.Volume
	}{
		{
			name:   "VolumeClaim",
			script: Script{Type: "VolumeClaim", Name: "test"},
			expected: []corev1.Volume{
				{
					Name: "k6-test-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test",
						},
					},
				},
			},
		},
		{
			name:   "ConfigMap",
			script: Script{Type: "ConfigMap", Name: "test"},
			expected: []corev1.Volume{
				{
					Name: "k6-test-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test",
							},
						},
					},
				},
			},
		},
		{
			name:     "no type (should be blocked by early validation)",
			script:   Script{Name: "test"},
			expected: []corev1.Volume{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := deep.Equal(tt.script.Volume(), tt.expected); diff != nil {
				t.Errorf("Volume() diff: %s", diff)
			}
		})
	}
}

func Test_Script_VolumeMount(t *testing.T) {
	tests := []struct {
		name     string
		script   Script
		expected []corev1.VolumeMount
	}{
		{
			name:   "ConfigMap always mounts at /test readonly",
			script: Script{Type: "ConfigMap", Name: "my-cm"},
			expected: []corev1.VolumeMount{
				{Name: "k6-test-volume", MountPath: "/test", ReadOnly: true},
			},
		},
		{
			name:   "VolumeClaim writable",
			script: Script{Type: "VolumeClaim", ReadOnly: false},
			expected: []corev1.VolumeMount{
				{Name: "k6-test-volume", ReadOnly: false},
			},
		},
		{
			name:   "VolumeClaim mounts at absolute path",
			script: Script{Type: "VolumeClaim", Path: "/data", ReadOnly: true},
			expected: []corev1.VolumeMount{
				{Name: "k6-test-volume", MountPath: "/data", ReadOnly: true},
			},
		},
		{
			name:   "VolumeClaim with relative path (blocked by earlier validation in practice)",
			script: Script{Type: "VolumeClaim", Path: "data/scripts"},
			expected: []corev1.VolumeMount{
				{Name: "k6-test-volume", MountPath: "data/scripts"},
			},
		},
		{
			name:   "VolumeClaim with no path (blocked by earlier validation in practice)",
			script: Script{Type: "VolumeClaim"},
			expected: []corev1.VolumeMount{
				{Name: "k6-test-volume", MountPath: ""},
			},
		},
		{
			name:     "LocalFile returns empty volumeMount",
			script:   Script{Type: "LocalFile"},
			expected: []corev1.VolumeMount{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := deep.Equal(tt.script.VolumeMount(), tt.expected); diff != nil {
				t.Errorf("VolumeMount() diff: %s", diff)
			}
		})
	}
}

func Test_Script_FullName(t *testing.T) {
	tests := []struct {
		name     string
		script   Script
		expected string
	}{
		{"path and filename present", Script{Path: "/test/", Filename: "script.js"}, "/test/script.js"},
		{"empty path", Script{Filename: "script.js"}, "script.js"},
		{"empty filename", Script{Path: "/test/"}, "/test/"},
		{"both empty", Script{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.script.FullName())
		})
	}
}

func Test_Script_UpdateCommand(t *testing.T) {
	baseCmd := []string{"k6", "run", "script.js"}

	tests := []struct {
		name   string
		script Script
		check  func(t *testing.T, result []string)
	}{
		{
			name:   "LocalFile wraps command with existence check",
			script: Script{Type: "LocalFile", Path: "/test/", Filename: "script.js"},
			check: func(t *testing.T, result []string) {
				assert.Equal(t, []string{"sh", "-c"}, result[:2])
				assert.Contains(t, result[2], "/test/script.js")
				assert.Contains(t, result[2], "LocalFile not found exiting...")
				assert.Contains(t, result[2], "k6 run script.js")
			},
		},
		{
			name:   "ConfigMap returns command unchanged",
			script: Script{Type: "ConfigMap"},
			check: func(t *testing.T, result []string) {
				assert.Equal(t, baseCmd, result)
			},
		},
		{
			name:   "VolumeClaim returns command unchanged",
			script: Script{Type: "VolumeClaim"},
			check: func(t *testing.T, result []string) {
				assert.Equal(t, baseCmd, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.script.UpdateCommand(baseCmd)
			tt.check(t, result)
		})
	}
}
