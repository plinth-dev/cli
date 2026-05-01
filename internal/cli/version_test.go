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
