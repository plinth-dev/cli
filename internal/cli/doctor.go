package cli

import (
	"flag"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const doctorUsage = `plinth doctor — check that your local toolchain can build the starters.

Required for the API starter:  go (>= 1.25), git
Required for the web starter:  node (>= 20), pnpm (>= 9), git
Optional:                      docker (for the local Postgres + Cerbos compose)

Flags:
  --verbose    show the version-detection command for each tool
`

// toolCheck describes a single tool the doctor probes.
type toolCheck struct {
	name        string
	required    bool
	versionArgs []string
	// versionRegex captures the semantic version (without leading 'v') from the
	// command's stdout/stderr.
	versionRegex *regexp.Regexp
	// minMajor / minMinor is the minimum acceptable version. Both zero disables
	// version comparison (presence-only check).
	minMajor int
	minMinor int
}

var doctorChecks = []toolCheck{
	{
		name:         "go",
		required:     true,
		versionArgs:  []string{"version"},
		versionRegex: regexp.MustCompile(`go(\d+)\.(\d+)`),
		minMajor:     1, minMinor: 25,
	},
	{
		name:         "git",
		required:     true,
		versionArgs:  []string{"--version"},
		versionRegex: regexp.MustCompile(`git version (\d+)\.(\d+)`),
		minMajor:     2, minMinor: 30,
	},
	{
		name:         "node",
		required:     true,
		versionArgs:  []string{"--version"},
		versionRegex: regexp.MustCompile(`v(\d+)\.(\d+)`),
		minMajor:     20, minMinor: 0,
	},
	{
		name:         "pnpm",
		required:     true,
		versionArgs:  []string{"--version"},
		versionRegex: regexp.MustCompile(`(\d+)\.(\d+)`),
		minMajor:     9, minMinor: 0,
	},
	{
		name:         "docker",
		required:     false,
		versionArgs:  []string{"--version"},
		versionRegex: regexp.MustCompile(`(\d+)\.(\d+)`),
	},
}

// doctorEnv lets tests stub out exec.LookPath / exec.Command.
type doctorEnv struct {
	lookPath func(string) (string, error)
	output   func(name string, args ...string) ([]byte, error)
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	return runDoctorWithEnv(args, stdout, stderr, doctorEnv{
		lookPath: exec.LookPath,
		output: func(name string, a ...string) ([]byte, error) {
			return exec.Command(name, a...).CombinedOutput()
		},
	})
}

func runDoctorWithEnv(args []string, stdout, stderr io.Writer, env doctorEnv) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, doctorUsage) }
	verbose := fs.Bool("verbose", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	failed := 0
	for _, c := range doctorChecks {
		status, detail := probe(c, env)
		fmt.Fprintf(stdout, "%-10s %s", c.name, status)
		if detail != "" {
			fmt.Fprintf(stdout, "  %s", detail)
		}
		if *verbose {
			fmt.Fprintf(stdout, "  [%s %s]", c.name, strings.Join(c.versionArgs, " "))
		}
		fmt.Fprintln(stdout)
		if status == "FAIL" && c.required {
			failed++
		}
	}

	fmt.Fprintln(stdout)
	if failed > 0 {
		fmt.Fprintf(stdout, "%d required tool(s) missing or below minimum version.\n", failed)
		return 1
	}
	fmt.Fprintln(stdout, "All required tools available.")
	return 0
}

func probe(c toolCheck, env doctorEnv) (status, detail string) {
	if _, err := env.lookPath(c.name); err != nil {
		if c.required {
			return "FAIL", "not found in PATH"
		}
		return "SKIP", "not found in PATH (optional)"
	}
	out, err := env.output(c.name, c.versionArgs...)
	if err != nil {
		return "FAIL", fmt.Sprintf("`%s %s` failed: %v", c.name, strings.Join(c.versionArgs, " "), err)
	}
	major, minor, raw, ok := parseVersion(c.versionRegex, string(out))
	if !ok {
		return "WARN", "could not parse version"
	}
	if c.minMajor == 0 && c.minMinor == 0 {
		return "OK", raw
	}
	if major < c.minMajor || (major == c.minMajor && minor < c.minMinor) {
		return "FAIL", fmt.Sprintf("found %s, need >= %d.%d", raw, c.minMajor, c.minMinor)
	}
	return "OK", raw
}

func parseVersion(re *regexp.Regexp, s string) (major, minor int, raw string, ok bool) {
	m := re.FindStringSubmatch(s)
	if len(m) < 3 {
		return 0, 0, "", false
	}
	a, err1 := strconv.Atoi(m[1])
	b, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0, "", false
	}
	return a, b, fmt.Sprintf("%d.%d", a, b), true
}
