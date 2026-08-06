package types

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseCLI(t *testing.T) {
	tests := []struct {
		name             string
		argv             []string
		cli              CLI
		invalidArguments bool
	}{
		{
			"EmptyArgs",
			nil,
			CLI{},
			false,
		},
		{
			"ShortArchiveArgs",
			[]string{"-u", "10", "-d", "5"},
			CLI{
				ArchiveArgs: []string{"-u", "10", "-d", "5"},
			},
			false,
		},
		{
			"LongArchiveArgs",
			[]string{"--vus", "10", "--duration", "5"},
			CLI{
				ArchiveArgs: []string{"--vus", "10", "--duration", "5"},
			},
			false,
		},
		{
			"ShortNonArchiveArg",
			[]string{"-u", "10", "-d", "5", "-l"},
			CLI{
				ArchiveArgs: []string{"-u", "10", "-d", "5"},
			},
			false,
		},
		{
			"LongNonArchiveArgs",
			[]string{"--vus", "10", "--duration", "5", "--linger"},
			CLI{
				ArchiveArgs: []string{"--vus", "10", "--duration", "5"},
			},
			false,
		},
		{
			"OutWithoutCloudArgs",
			[]string{"--vus", "10", "-o", "json", "-o", "csv"},
			CLI{
				ArchiveArgs: []string{"--vus", "10"},
				HasCloudOut: false,
			},
			false,
		},
		{
			"OutWithCloudArgs",
			[]string{"--vus", "10", "--out", "json", "-o", "csv", "--out", "cloud"},
			CLI{
				ArchiveArgs: []string{"--vus", "10"},
				HasCloudOut: true,
			},
			false,
		},
		{
			"VerboseOutWithCloudArgs",
			[]string{"--vus", "10", "--out", "json", "-o", "csv", "--out", "cloud", "--verbose"},
			CLI{
				ArchiveArgs: []string{"--vus", "10"},
				HasCloudOut: true,
			},
			false,
		},
		{
			"FalseCloudInTag",
			[]string{"--tag", "cloud", "--out", "json"},
			CLI{
				ArchiveArgs: []string{"--tag", "cloud"},
				HasCloudOut: false,
			},
			false,
		},
		{
			"StandaloneCloudOut",
			[]string{"--out", "cloud"},
			CLI{
				HasCloudOut: true,
			},
			false,
		},
		{
			"StandaloneShortCloudOut",
			[]string{"-o", "cloud"},
			CLI{
				HasCloudOut: true,
			},
			false,
		},
		{
			// with `.spec.arguments`, the value is split and has quotes
			"OmitLogOutputArguments",
			[]string{
				"--out", "cloud", "--no-thresholds",
				`--log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=my-plz,label.test_run_id=1111,header.Authorization="Token`,
				`$(K6_CLOUD_TOKEN)"`,
			},
			CLI{
				ArchiveArgs: []string{"--no-thresholds"},
				HasCloudOut: true,
			},
			false,
		},
		{
			// with `.spec.arguments`, the value is split and has quotes
			"OmitLogOutputAgumentsInDiffOrder",
			[]string{
				"--out", "cloud",
				`--log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=my-plz,label.test_run_id=1111,header.Authorization="Token`,
				`$(K6_CLOUD_TOKEN)"`,
				"--no-thresholds",
			},
			CLI{
				ArchiveArgs: []string{"--no-thresholds"},
				HasCloudOut: true,
			},
			false,
		},
		{
			// with `.spec.args`, the value is kept together, without quotes
			"OmitLogOutputArgs",
			[]string{
				"--out", "cloud",
				`--log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=my-plz,label.test_run_id=1111,header.Authorization=Token $(K6_CLOUD_TOKEN)`,
				"--no-thresholds",
			},
			CLI{
				ArchiveArgs: []string{"--no-thresholds"},
				HasCloudOut: true,
			},
			false,
		},
		{
			"InvalidArguments",
			[]string{"run", "this-argument-does-not-matter.js", "-o", "json"},
			CLI{},
			true,
		},
		{
			"SkipBlockHostnamesEquals",
			[]string{"--vus", "10", `--block-hostnames="google.com"`, "--duration", "5s"},
			CLI{
				ArchiveArgs: []string{"--vus", "10", "--duration", "5s"},
			},
			false,
		},
		{
			"SkipBlacklistIpEquals",
			[]string{"--vus", "10", `--blacklist-ip="8.8.8.8/32"`, "--duration", "5s"},
			CLI{
				ArchiveArgs: []string{"--vus", "10", "--duration", "5s"},
			},
			false,
		},
		{
			"SkipUserAgentEquals",
			[]string{"--vus", "10", `--user-agent="foo"`, "--duration", "5s"},
			CLI{
				ArchiveArgs: []string{"--vus", "10", "--duration", "5s"},
			},
			false,
		},
		{
			"KeepKubeEnvRefs",
			[]string{"--out", "cloud", "-e", "FOO=$(K6_OP_ENV_FOO)", "--no-thresholds", "-e", "BAR=plain"},
			CLI{
				ArchiveArgs: []string{"-e", "FOO=$(K6_OP_ENV_FOO)", "--no-thresholds", "-e", "BAR=plain"},
				HasCloudOut: true,
			},
			false,
		},
		{
			"IncludeSystemEnvVars",
			[]string{"--out", "cloud", "--include-system-env-vars"},
			CLI{
				ArchiveArgs: []string{"--include-system-env-vars"},
				HasCloudOut: true,
			},
			false,
		},
		{
			// valid for .spec.args only
			"ExactSpacedAndQuotedElements",
			[]string{"--tag", "note=hello world", "-e", `QUOTED="value"`},
			CLI{
				ArchiveArgs: []string{"--tag", "note=hello world", "-e", `QUOTED="value"`},
			},
			false,
		},
		{
			// valid for .spec.args only
			"ExactDollarsAreNotInterpreted",
			[]string{"--tag", "price=$$100 $(NAME)"},
			CLI{
				ArchiveArgs: []string{"--tag", "price=$$100 $(NAME)"},
			},
			false,
		},
		{
			"EmptyElement",
			[]string{"", "--vus", "", "10"},
			CLI{
				ArchiveArgs: []string{"--vus", "10"},
			},
			false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			original := slices.Clone(test.argv)

			cli, err := ParseCLI(test.argv)
			assert.Equal(t, test.invalidArguments, err != nil)
			assert.Equal(t, test.cli.ArchiveArgs, cli.ArchiveArgs)
			assert.Equal(t, test.cli.HasCloudOut, cli.HasCloudOut)
			assert.Equal(t, original, test.argv, "ParseCLI must not modify its input")
		})
	}
}

func Test_KubeExpansionToShell(t *testing.T) {
	assert.Equal(t,
		`-e FOO="${K6_OP_ENV_FOO}" -e BAR=plain`,
		KubeExpansionToShell(`-e FOO=$(K6_OP_ENV_FOO) -e BAR=plain`),
	)
}
