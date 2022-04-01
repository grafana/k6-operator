package types

import (
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/k6-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// Internal type created to support Spec.script options
type Script struct {
	Name string
	File string
	Type string // ConfigMap | VolumeClaim | LocalFile
}

// ParseScript extracts Script data bits from K6 spec and performs basic validation
func ParseScript(spec *v1alpha1.K6Spec) (*Script, error) {
	s := &Script{}
	s.File = "test.js"

	if spec.Script.VolumeClaim.Name != "" {
		s.Name = spec.Script.VolumeClaim.Name
		if spec.Script.VolumeClaim.File != "" {
			s.File = spec.Script.VolumeClaim.File
		}

		s.File = fmt.Sprintf("/test/%s", s.File)
		s.Type = "VolumeClaim"
		return s, nil
	}

	if spec.Script.ConfigMap.Name != "" {
		s.Name = spec.Script.ConfigMap.Name

		if spec.Script.ConfigMap.File != "" {
			s.File = spec.Script.ConfigMap.File
		}

		s.File = fmt.Sprintf("/test/%s", s.File)
		s.Type = "ConfigMap"
		return s, nil
	}

	if spec.Script.LocalFile != "" {
		s.Name = "LocalFile"
		s.File = spec.Script.LocalFile
		s.Type = "LocalFile"
		return s, nil
	}

	return nil, errors.New("Script definition should contain one of: ConfigMap, VolumeClaim, LocalFile")
}

// Volume creates a Volume spec for the script
func (s *Script) Volume() corev1.Volume {
	switch s.Type {
	case "VolumeClaim":
		return corev1.Volume{
			Name: "k6-test-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: s.Name,
				},
			},
		}

	case "ConfigMap":
		return corev1.Volume{
			Name: "k6-test-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: s.Name,
					},
				},
			},
		}
	default:
		return corev1.Volume{}
	}
}

// VolumeMount creates a VolumeMount spec for the script
func (s *Script) VolumeMount() corev1.VolumeMount {
	if s.Type == "LocalFile" {
		return corev1.VolumeMount{}
	}
	return corev1.VolumeMount{
		Name:      "k6-test-volume",
		MountPath: "/test",
	}
}

// UpdateCommand modifies command to check for script existence in case of LocalFile;
// otherwise, command remains unmodified
func (s *Script) UpdateCommand(cmd []string) []string {
	if s.Type == "LocalFile" {
		joincmd := strings.Join(cmd, " ")
		checkCommand := []string{"sh", "-c", fmt.Sprintf("if [ ! -f %v ]; then echo \"LocalFile not found exiting...\"; exit 1; fi;\n%v", s.File, joincmd)}
		return checkCommand
	}
	return cmd
}

// Internal type to support k6 invocation in initialization stage.
// Not all k6 commands allow the same set of arguments so CLI is object meant to contain only the ones fit for the arhive call.
// Maybe revise this once crococonf is closer to integration?
type CLI struct {
	ArchiveArgs string
	// k6-operator doesn't care for most values of CLI arguments to k6, with an exception of cloud output
	HasCloudOut bool
}

func ParseCLI(spec *v1alpha1.K6Spec) *CLI {
	lastArgV := func(start int, args []string) (end int) {
		var nextArg bool
		end = start + 1
		for !nextArg && end < len(args) {
			if args[end][0] == '-' {
				nextArg = true
				break
			}
			end++
		}
		return
	}

	var cli CLI

	args := strings.Split(spec.Arguments, " ")
	i := 0
	for i < len(args) {
		args[i] = strings.TrimSpace(args[i])
		if len(args[i]) == 0 {
			i++
			continue
		}
		if args[i][0] == '-' {
			end := lastArgV(i+1, args)

			switch args[i] {
			case "-o", "--out":
				for j := 0; j < end; j++ {
					if args[j] == "cloud" {
						cli.HasCloudOut = true
					}
				}
			case "-l", "--linger", "--no-usage-report":
				// non-archive arguments, so skip them
				break
			default:
				if len(cli.ArchiveArgs) > 0 {
					cli.ArchiveArgs += " "
				}
				cli.ArchiveArgs += strings.Join(args[i:end], " ")
			}
			i = end
		}
	}

	return &cli
}
