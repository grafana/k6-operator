package types

import (
	"testing"

	deep "github.com/go-test/deep"
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
