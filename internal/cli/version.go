package cli

import (
	"fmt"
	"io"
	"runtime"
)

// Set via -ldflags "-X github.com/plinth-dev/cli/internal/cli.Version=..." at build.
var (
	Version = "dev"
	Commit  = "none"
)

func runVersion(stdout io.Writer) int {
	fmt.Fprintf(stdout, "plinth %s (commit %s, %s)\n", Version, Commit, runtime.Version())
	return 0
}
