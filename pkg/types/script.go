package types

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// Internal type created to support Spec.script options
type Script struct {
	Name     string // Name of ConfigMap or VolumeClaim or "LocalFile"
	ReadOnly bool   // VolumeClaim only
	Filename string
	Path     string
	Type     string // ConfigMap | VolumeClaim | LocalFile
}

func (s *Script) FullName() string {
	return s.Path + s.Filename
}

// Volume creates a Volume spec for the script
func (s *Script) Volume() []corev1.Volume {
	switch s.Type {
	case "VolumeClaim":
		return []corev1.Volume{
			corev1.Volume{
				Name: "k6-test-volume",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: s.Name,
						ReadOnly:  s.ReadOnly,
					},
				},
			},
		}

	case "ConfigMap":
		return []corev1.Volume{
			corev1.Volume{
				Name: "k6-test-volume",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: s.Name,
						},
					},
				},
			},
		}

	default:
		return []corev1.Volume{}
	}
}

// VolumeMount creates a VolumeMount spec for the script
func (s *Script) VolumeMount() []corev1.VolumeMount {
	if s.Type == "LocalFile" {
		return []corev1.VolumeMount{}
	}

	return []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      "k6-test-volume",
			MountPath: "/test",
		},
	}
}

// UpdateCommand modifies command to check for script existence in case of LocalFile;
// otherwise, command remains unmodified
func (s *Script) UpdateCommand(cmd []string) []string {
	if s.Type == "LocalFile" {
		joincmd := strings.Join(cmd, " ")
		checkCommand := []string{
			"sh",
			"-c",
			fmt.Sprintf("if [ ! -f %v ]; then echo \"LocalFile not found exiting...\"; exit 1; fi;\n%v", s.FullName(), joincmd),
		}
		return checkCommand
	}
	return cmd
}
