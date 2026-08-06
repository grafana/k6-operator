package types

import (
	"fmt"
	"strings"
)

// CLI is an internal type to support k6 invocation in initialization stage.
// Not all k6 commands allow the same set of arguments so CLI is an object
// meant to contain only the ones fit for the archive call.
// Maybe revise this once crococonf is closer to integration?
type CLI struct {
	ArchiveArgs []string
	// k6-operator doesn't care for most values of CLI arguments to k6, with an exception of cloud output
	HasCloudOut bool
}

// ParseCLI parses the k6 arguments and returns the subset of them that is
// suitable for the `k6 archive` call.
func ParseCLI(argv []string) (*CLI, error) {
	// find the end of value for last argument
	lastArgV := func(start int, args []string) (end int) {
		end = start
		for end < len(args) {
			if len(args[end]) == 0 {
				// An empty element right after a flag is its value, e.g. `--user-agent ""`:
				// k6 CLI allows it, so keep the pairing intact. An empty element anywhere
				// else is positional: don't consume it here, so that the main loop errors
				// on it; just like k6 CLI errors on an extra empty positional argument.
				if end == start {
					end++
				}
				break
			}
			if args[end][0] == '-' {
				break
			}
			end++
		}
		return
	}

	var (
		cli CLI
		err error
	)

	i := 0
	for i < len(argv) {
		if len(argv[i]) > 0 && argv[i][0] == '-' {
			end := lastArgV(i+1, argv)

			switch {
			case strings.HasPrefix(argv[i], "--log-output"):
				// `k6 archive` ignores this argument but if it contains an env var
				// for token (may be the case for PLZ test runs), it will break the shell;
				// so omit it.

			case strings.HasPrefix(argv[i], "--block-hostnames"),
				strings.HasPrefix(argv[i], "--blacklist-ip"),
				strings.HasPrefix(argv[i], "--user-agent"):
				// Unsupported by `k6 archive`.

			case argv[i] == "-o", argv[i] == "--out":
				// "cloud" should appear after -o / --out
				// (Historically) Supported forms: --out cloud, -o cloud
				if i+1 < end && argv[i+1] == "cloud" {
					cli.HasCloudOut = true
				}

			case argv[i] == "-l", argv[i] == "--linger", argv[i] == "--no-usage-report":
				// non-archive arguments, so skip them

			case argv[i] == "--verbose", argv[i] == "-v":
				// this argument is acceptable by archive but it'd
				// mess up the JSON output of `k6 inspect`

			default:
				cli.ArchiveArgs = append(cli.ArchiveArgs, argv[i:end]...)
			}
			i = end
		} else {
			err = fmt.Errorf("encountered an invalid value for k6 CLI argument: `%s`", argv[i])
			break
		}
	}

	return &cli, err
}
