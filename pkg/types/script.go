package types

import (
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

	switch s.Type {

	// VolumeClaim: mounts the volume at s.Path (default "/test").
	case "VolumeClaim":
		return []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      "k6-test-volume",
				MountPath: s.Path,
				ReadOnly:  s.ReadOnly,
			},
		}

	// ConfigMap: always mounted at "/test" since keys cannot represent nested directories.
	case "ConfigMap":
		return []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      "k6-test-volume",
				MountPath: "/test",
				ReadOnly:  true,
			},
		}

	default: // LocalFile
		return []corev1.VolumeMount{}

	}

}

const localFileCheckScript = `if [ ! -f "$1" ]; then
  echo "LocalFile not found exiting..."
  exit 1
fi
shift
exec "$@"`

// UpdateCommand modifies command to check for script existence in case of LocalFile;
// otherwise, command remains unmodified.
// The command, when passed to the existence check, can take two forms:
// - with .spec.arguments, via shell interpreter `sh -c` and concat of arguments
// - with .spec.args, via shell positional arguments (more correct and safer)
func (s *Script) UpdateCommand(cmd []string, needsShellCmd bool) []string {
	if s.Type != "LocalFile" {
		return cmd
	}

	// The command from .spec.arguments needs shell interpretation.
	// This is a backwards compatible approach.
	if needsShellCmd {
		cmd = []string{"sh", "-c", strings.Join(cmd, " ")}
	}

	// Second `sh` is a placeholder for $0 (unused in the existence check, so doesn't matter)
	// Script name ($1) is always a positional argument.
	checkCommand := []string{"sh", "-c", localFileCheckScript, "sh", s.FullName()}
	return append(checkCommand, cmd...)
}
