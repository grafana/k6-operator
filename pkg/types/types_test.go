package types

import (
	"testing"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func Test_ParseCLI(t *testing.T) {
	tests := []struct {
		name    string
		argLine string
		cli     CLI
	}{
		{
			"EmptyArgs",
			"",
			CLI{},
		},
		{
			"ShortArchiveArgs",
			"-u 10 -d 5",
			CLI{
				ArchiveArgs: "-u 10 -d 5",
			},
		},
		{
			"LongArchiveArgs",
			"--vus 10 --duration 5",
			CLI{
				ArchiveArgs: "--vus 10 --duration 5",
			},
		},
		{
			"ShortNonArchiveArg",
			"-u 10 -d 5 -l",
			CLI{
				ArchiveArgs: "-u 10 -d 5",
			},
		},
		{
			"LongNonArchiveArgs",
			"--vus 10 --duration 5 --linger",
			CLI{
				ArchiveArgs: "--vus 10 --duration 5",
			},
		},
		{
			"OutWithoutCloudArgs",
			"--vus 10 -o json -o csv",
			CLI{
				ArchiveArgs: "--vus 10",
				HasCloudOut: false,
			},
		},
		{
			"OutWithCloudArgs",
			"--vus 10 --out json -o csv --out cloud",
			CLI{
				ArchiveArgs: "--vus 10",
				HasCloudOut: true,
			},
		},
		{
			"VerboseOutWithCloudArgs",
			"--vus 10 --out json -o csv --out cloud --verbose",
			CLI{
				ArchiveArgs: "--vus 10",
				HasCloudOut: true,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			spec := v1alpha1.K6Spec{
				Arguments: test.argLine,
			}
			cli := ParseCLI(&spec)

			assert.Equal(t, test.cli.ArchiveArgs, cli.ArchiveArgs)
			assert.Equal(t, test.cli.HasCloudOut, cli.HasCloudOut)
		})
	}
}
