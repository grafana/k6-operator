package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseCLI(t *testing.T) {
	tests := []struct {
		name             string
		argLine          string
		cli              CLI
		invalidArguments bool
	}{
		{
			"EmptyArgs",
			"",
			CLI{},
			false,
		},
		{
			"ShortArchiveArgs",
			"-u 10 -d 5",
			CLI{
				ArchiveArgs: "-u 10 -d 5",
			},
			false,
		},
		{
			"LongArchiveArgs",
			"--vus 10 --duration 5",
			CLI{
				ArchiveArgs: "--vus 10 --duration 5",
			},
			false,
		},
		{
			"ShortNonArchiveArg",
			"-u 10 -d 5 -l",
			CLI{
				ArchiveArgs: "-u 10 -d 5",
			},
			false,
		},
		{
			"LongNonArchiveArgs",
			"--vus 10 --duration 5 --linger",
			CLI{
				ArchiveArgs: "--vus 10 --duration 5",
			},
			false,
		},
		{
			"OutWithoutCloudArgs",
			"--vus 10 -o json -o csv",
			CLI{
				ArchiveArgs: "--vus 10",
				HasCloudOut: false,
			},
			false,
		},
		{
			"OutWithCloudArgs",
			"--vus 10 --out json -o csv --out cloud",
			CLI{
				ArchiveArgs: "--vus 10",
				HasCloudOut: true,
			},
			false,
		},
		{
			"VerboseOutWithCloudArgs",
			"--vus 10 --out json -o csv --out cloud --verbose",
			CLI{
				ArchiveArgs: "--vus 10",
				HasCloudOut: true,
			},
			false,
		},
		{
			"OmitLogOutput",
			`--out cloud --no-thresholds --log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=my-plz,label.test_run_id=1111,header.Authorization="Token $(K6_CLOUD_TOKEN)"`,
			CLI{
				ArchiveArgs: "--no-thresholds",
				HasCloudOut: true,
			},
			false,
		},
		{
			"OmitLogOutputInDiffOrder",
			`--out cloud --log-output=loki=https://cloudlogs.k6.io/api/v1/push,label.lz=my-plz,label.test_run_id=1111,header.Authorization="Token $(K6_CLOUD_TOKEN)" --no-thresholds`,
			CLI{
				ArchiveArgs: "--no-thresholds",
				HasCloudOut: true,
			},
			false,
		},
		{
			"InvalidArguments",
			`run this-argument-does-not-matter.js -o json`,
			CLI{},
			true,
		},
		{
			"SkipBlockHostnamesEquals",
			`--vus 10 --block-hostnames="google.com" --duration 5s`,
			CLI{
				ArchiveArgs: "--vus 10 --duration 5s",
			},
			false,
		},
		{
			"SkipBlacklistIpEquals",
			`--vus 10 --blacklist-ip="8.8.8.8/32" --duration 5s`,
			CLI{
				ArchiveArgs: "--vus 10 --duration 5s",
			},
			false,
		},
		{
			"SkipUserAgentEquals",
			`--vus 10 --user-agent="foo" --duration 5s`,
			CLI{
				ArchiveArgs: "--vus 10 --duration 5s",
			},
			false,
		},
		{
			"IncludeSystemEnvVars",
			`--out cloud --include-system-env-vars`,
			CLI{
				ArchiveArgs: "--include-system-env-vars",
				HasCloudOut: true,
			},
			false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cli, err := ParseCLI(test.argLine)
			assert.Equal(t, test.invalidArguments, err != nil)
			assert.Equal(t, test.cli.ArchiveArgs, cli.ArchiveArgs)
			assert.Equal(t, test.cli.HasCloudOut, cli.HasCloudOut)
		})
	}
}
