package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionPrints(t *testing.T) {
	defer func(v, c string) { Version, Commit = v, c }(Version, Commit)
	Version = "0.1.0"
	Commit = "abc123"

	var stdout bytes.Buffer
	if code := runVersion(&stdout); code != 0 {
		t.Fatalf("exit=%d", code)
	}
	got := stdout.String()
	if !strings.Contains(got, "plinth 0.1.0") {
		t.Errorf("missing version: %s", got)
	}
	if !strings.Contains(got, "commit abc123") {
		t.Errorf("missing commit: %s", got)
	}
}

func TestResolveVersionPrefersExplicit(t *testing.T) {
	v, c := resolveVersion("0.2.0", "deadbeef")
	if v != "0.2.0" || c != "deadbeef" {
		t.Errorf("explicit args lost: v=%q c=%q", v, c)
	}
}

func TestResolveVersionFallback(t *testing.T) {
	// In `go test` runs, debug.ReadBuildInfo reports Main.Version="(devel)"
	// and no vcs.revision setting, so both should land on the dev/none defaults.
	v, c := resolveVersion("", "")
	if v != "dev" {
		t.Errorf("unset version did not fall back to dev: %q", v)
	}
	if c == "" {
		t.Errorf("commit must not be empty after fallback")
	}
}
