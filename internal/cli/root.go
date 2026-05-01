// Package cli implements the plinth subcommand dispatch.
package cli

import (
	"fmt"
	"io"
)

const usage = `plinth — scaffold modules from the Plinth starters.

Usage:
  plinth <command> [flags]

Commands:
  new       Scaffold a new module from starter-web and/or starter-api
  doctor    Check that local toolchain prerequisites are installed
  version   Print the CLI version

Run "plinth <command> --help" for command-specific flags.
`

// Run dispatches a subcommand and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "new":
		return runNew(rest, stdout, stderr)
	case "doctor":
		return runDoctor(rest, stdout, stderr)
	case "version", "--version", "-v":
		return runVersion(stdout)
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "plinth: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
}
