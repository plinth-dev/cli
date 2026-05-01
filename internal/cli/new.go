package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/plinth-dev/cli/internal/fetch"
	"github.com/plinth-dev/cli/internal/rename"
)

const newUsage = `plinth new — scaffold a new module from the Plinth starters.

Usage:
  plinth new <name> [flags]

  <name> must be lowercase kebab-case (e.g. "billing", "user-profile").
  By default both --web and --api scaffolds are created. Use --web or --api
  alone to scaffold only one.

  Web scaffold lands in   <dir>/<name>-web/
  API scaffold lands in   <dir>/<name>-api/

Flags:
  --web                 scaffold the Next.js starter (default if --api unset)
  --api                 scaffold the Go starter (default if --web unset)
  --dir DIR             parent directory for the new scaffolds (default ".")
  --module-path PATH    Go module path for the API scaffold
                          (default "github.com/example/<name>-api")
  --ref REF             starter tag to fetch (default "v0.1.0")
  --no-git              skip "git init" inside generated directories

Example:
  plinth new billing --module-path github.com/acme/billing-api
`

// scaffoldDeps lets tests inject a fake fetcher and skip the real git binary.
type scaffoldDeps struct {
	fetcher  starterFetcher
	gitInit  func(dir string) error
}

type starterFetcher interface {
	FetchAndExtract(ctx context.Context, owner, repo, ref, dst string) error
}

func runNew(args []string, stdout, stderr io.Writer) int {
	return runNewWithDeps(args, stdout, stderr, scaffoldDeps{
		fetcher: fetch.New(),
		gitInit: defaultGitInit,
	})
}

func runNewWithDeps(args []string, stdout, stderr io.Writer, deps scaffoldDeps) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, newUsage) }

	var (
		web        = fs.Bool("web", false, "")
		api        = fs.Bool("api", false, "")
		dir        = fs.String("dir", ".", "")
		modulePath = fs.String("module-path", "", "")
		ref        = fs.String("ref", "v0.1.0", "")
		noGit      = fs.Bool("no-git", false, "")
	)

	name, rest, ok := extractName(args)
	if !ok {
		fmt.Fprint(stderr, newUsage)
		return 2
	}
	if err := fs.Parse(rest); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "plinth: unexpected extra arguments: %v\n", fs.Args())
		return 2
	}
	if !validName(name) {
		fmt.Fprintf(stderr, "plinth: invalid name %q — must be lowercase kebab-case (e.g. \"billing\")\n", name)
		return 2
	}

	doWeb, doAPI := *web, *api
	if !doWeb && !doAPI {
		doWeb, doAPI = true, true
	}

	if *modulePath == "" {
		*modulePath = "github.com/example/" + name + "-api"
	}

	parent, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "plinth: resolve --dir: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		fmt.Fprintf(stderr, "plinth: mkdir %s: %v\n", parent, err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	type plan struct {
		repo         string
		dst          string
		serviceName  string
		replacements []rename.Replacement
	}
	var plans []plan
	if doAPI {
		serviceName := name + "-api"
		plans = append(plans, plan{
			repo:         "starter-api",
			dst:          filepath.Join(parent, serviceName),
			serviceName:  serviceName,
			replacements: rename.ForAPI(*modulePath, serviceName),
		})
	}
	if doWeb {
		serviceName := name + "-web"
		plans = append(plans, plan{
			repo:         "starter-web",
			dst:          filepath.Join(parent, serviceName),
			serviceName:  serviceName,
			replacements: rename.ForWeb(serviceName),
		})
	}

	for _, p := range plans {
		if _, err := os.Stat(p.dst); err == nil {
			fmt.Fprintf(stderr, "plinth: refusing to scaffold into existing path %s\n", p.dst)
			return 1
		} else if !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "plinth: stat %s: %v\n", p.dst, err)
			return 1
		}
	}

	for _, p := range plans {
		fmt.Fprintf(stdout, "→ fetching %s@%s into %s\n", p.repo, *ref, p.dst)
		if err := deps.fetcher.FetchAndExtract(ctx, "plinth-dev", p.repo, *ref, p.dst); err != nil {
			fmt.Fprintf(stderr, "plinth: %v\n", err)
			_ = os.RemoveAll(p.dst)
			return 1
		}
		fmt.Fprintf(stdout, "→ rewriting identifiers in %s\n", p.serviceName)
		if err := rename.Apply(p.dst, p.replacements); err != nil {
			fmt.Fprintf(stderr, "plinth: %v\n", err)
			return 1
		}
		if !*noGit {
			if err := deps.gitInit(p.dst); err != nil {
				fmt.Fprintf(stderr, "plinth: warning: git init failed in %s: %v\n", p.dst, err)
			}
		}
	}

	printNextSteps(stdout, name, parent, doWeb, doAPI, *modulePath)
	return 0
}

// extractName pulls the first non-flag argument out of args. It treats `-x` and
// `--x` as flag tokens; if a flag uses the space-separated form (`--dir foo`)
// the value `foo` would look like a positional, so we also skip the next token
// for known value-taking flags.
func extractName(args []string) (string, []string, bool) {
	valueFlags := map[string]bool{
		"--dir": true, "--module-path": true, "--ref": true,
		"-dir": true, "-module-path": true, "-ref": true,
	}
	rest := make([]string, 0, len(args))
	skipNext := false
	name := ""
	for _, a := range args {
		if skipNext {
			rest = append(rest, a)
			skipNext = false
			continue
		}
		if strings.HasPrefix(a, "-") {
			rest = append(rest, a)
			if strings.Contains(a, "=") {
				continue
			}
			if valueFlags[a] {
				skipNext = true
			}
			continue
		}
		if name == "" {
			name = a
			continue
		}
		rest = append(rest, a)
	}
	if name == "" {
		return "", nil, false
	}
	return name, rest, true
}

var nameRE = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

func validName(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	return nameRE.MatchString(s)
}

func defaultGitInit(dir string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return err
	}
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = dir
	return cmd.Run()
}

func printNextSteps(w io.Writer, name, parent string, doWeb, doAPI bool, modulePath string) {
	display := func(sub string) string {
		full := filepath.Join(parent, sub)
		if rel, err := filepath.Rel(mustWD(), full); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
		return full
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Done. Next steps:")
	if doAPI {
		fmt.Fprintf(w, "  cd %s\n", display(name+"-api"))
		fmt.Fprintf(w, "    # Go module path is %s\n", modulePath)
		fmt.Fprintln(w, "    # Edit cerbos/policies/items.yaml to rename the resource kind")
		fmt.Fprintln(w, "    docker compose up -d")
		fmt.Fprintln(w, "    go run ./cmd/server")
		fmt.Fprintln(w, "")
	}
	if doWeb {
		fmt.Fprintf(w, "  cd %s\n", display(name+"-web"))
		fmt.Fprintln(w, "    pnpm install")
		fmt.Fprintln(w, "    pnpm dev")
	}
}

func mustWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return strings.TrimSpace(wd)
}
