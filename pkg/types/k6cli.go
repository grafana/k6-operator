package types

import "strings"

// CLI is an internal type to support k6 invocation in initialization stage.
// Not all k6 commands allow the same set of arguments so CLI is an object
// meant to contain only the ones fit for the archive call.
// Maybe revise this once crococonf is closer to integration?
type CLI struct {
	ArchiveArgs string
	// k6-operator doesn't care for most values of CLI arguments to k6, with an exception of cloud output
	HasCloudOut bool
}

func ParseCLI(arguments string) *CLI {
	lastArgV := func(start int, args []string) (end int) {
		end = start
		for end < len(args) {
			args[end] = strings.TrimSpace(args[end])
			if len(args[end]) > 0 && args[end][0] == '-' {
				break
			}
			end++
		}
		return
	}

	var cli CLI

	args := strings.Split(arguments, " ")
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
			case "--verbose", "-v":
				// this argument is acceptable by archive but it'd
				// mess up the JSON output of `k6 inspect`
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
