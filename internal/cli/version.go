package cli

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
)

// Set via -ldflags "-X github.com/plinth-dev/cli/internal/cli.Version=..." at
// build (see Makefile). When unset — e.g. when the user installs via
// `go install ...@v0.1.0` — runtime/debug.ReadBuildInfo provides the module
// version and VCS revision Go's toolchain stamped into the binary.
var (
	Version = ""
	Commit  = ""
)

func runVersion(stdout io.Writer) int {
	v, c := resolveVersion(Version, Commit)
	fmt.Fprintf(stdout, "plinth %s (commit %s, %s)\n", v, c, runtime.Version())
	return 0
}

func resolveVersion(version, commit string) (string, string) {
	if version != "" && commit != "" {
		return version, commit
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fallback(version, "dev"), fallback(commit, "none")
	}
	if version == "" {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			version = v
		}
	}
	if commit == "" {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				commit = s.Value
				if len(commit) > 12 {
					commit = commit[:12]
				}
				break
			}
		}
	}
	return fallback(version, "dev"), fallback(commit, "none")
}

func fallback(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
