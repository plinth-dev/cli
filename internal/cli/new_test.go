package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/plinth-dev/cli/internal/fetch"
)

func TestRunNew_BothScaffolds(t *testing.T) {
	srv := starterServer(t, map[string]map[string]string{
		"starter-api": {
			"go.mod":             "module github.com/plinth-dev/starter-api\n\ngo 1.25.0\n",
			"cmd/server/main.go": "package main\n// starter-api\n",
		},
		"starter-web": {
			"package.json":              `{"name": "starter-web"}`,
			"instrumentation-client.ts": `serviceName: "starter-web"`,
		},
	})
	defer srv.Close()

	tmp := t.TempDir()
	deps := scaffoldDeps{
		fetcher: &fetch.Client{BaseURL: srv.URL, HTTPClient: srv.Client()},
		gitInit: func(string) error { return nil },
	}

	var stdout, stderr bytes.Buffer
	code := runNewWithDeps(
		[]string{"billing", "--dir", tmp, "--module-path", "github.com/acme/billing-api", "--ref", "v0.1.0"},
		&stdout, &stderr, deps,
	)
	if code != 0 {
		t.Fatalf("runNew exit=%d stderr=%s", code, stderr.String())
	}

	gomod := mustRead(t, tmp, "billing-api/go.mod")
	if !strings.Contains(gomod, "module github.com/acme/billing-api") {
		t.Errorf("api go.mod not rewritten: %s", gomod)
	}
	main := mustRead(t, tmp, "billing-api/cmd/server/main.go")
	if !strings.Contains(main, "// billing-api") {
		t.Errorf("api main.go not rewritten: %s", main)
	}

	pkg := mustRead(t, tmp, "billing-web/package.json")
	if !strings.Contains(pkg, `"name": "billing-web"`) {
		t.Errorf("web package.json not rewritten: %s", pkg)
	}
}

func TestRunNew_WebOnly(t *testing.T) {
	srv := starterServer(t, map[string]map[string]string{
		"starter-web": {"package.json": `{"name": "starter-web"}`},
	})
	defer srv.Close()

	tmp := t.TempDir()
	deps := scaffoldDeps{
		fetcher: &fetch.Client{BaseURL: srv.URL, HTTPClient: srv.Client()},
		gitInit: func(string) error { return nil },
	}

	var stdout, stderr bytes.Buffer
	code := runNewWithDeps([]string{"acme", "--web", "--dir", tmp}, &stdout, &stderr, deps)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, "acme-api")); !os.IsNotExist(err) {
		t.Errorf("acme-api should not exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "acme-web")); err != nil {
		t.Errorf("acme-web should exist: %v", err)
	}
}

func TestRunNew_RejectsBadName(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runNewWithDeps([]string{"Bad_Name", "--dir", tmp}, &stdout, &stderr,
		scaffoldDeps{fetcher: noopFetcher{}, gitInit: func(string) error { return nil }})
	if code != 2 {
		t.Errorf("expected exit 2 for bad name, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid name") {
		t.Errorf("stderr missing message: %s", stderr.String())
	}
}

func TestRunNew_MissingName(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runNewWithDeps([]string{}, &stdout, &stderr,
		scaffoldDeps{fetcher: noopFetcher{}, gitInit: func(string) error { return nil }})
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestRunNew_RefusesExistingDir(t *testing.T) {
	srv := starterServer(t, map[string]map[string]string{
		"starter-api": {"go.mod": "module github.com/plinth-dev/starter-api\n"},
	})
	defer srv.Close()

	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "billing-api"), 0o755); err != nil {
		t.Fatal(err)
	}

	deps := scaffoldDeps{
		fetcher: &fetch.Client{BaseURL: srv.URL, HTTPClient: srv.Client()},
		gitInit: func(string) error { return nil },
	}
	var stdout, stderr bytes.Buffer
	code := runNewWithDeps([]string{"billing", "--api", "--dir", tmp}, &stdout, &stderr, deps)
	if code == 0 {
		t.Errorf("expected nonzero exit when target exists")
	}
	if !strings.Contains(stderr.String(), "existing path") {
		t.Errorf("stderr missing collision message: %s", stderr.String())
	}
}

func TestValidName(t *testing.T) {
	good := []string{"a", "billing", "user-profile", "x1", "abc-def-ghi", "v2"}
	bad := []string{"", "Billing", "user_profile", "-billing", "billing-", "1abc", "billing/api", strings.Repeat("a", 65)}
	for _, s := range good {
		if !validName(s) {
			t.Errorf("validName(%q) = false, want true", s)
		}
	}
	for _, s := range bad {
		if validName(s) {
			t.Errorf("validName(%q) = true, want false", s)
		}
	}
}

// noopFetcher satisfies starterFetcher for tests that error out before fetch.
type noopFetcher struct{}

func (noopFetcher) FetchAndExtract(_ context.Context, _, _, _, _ string) error {
	return nil
}

// starterServer returns an httptest.Server that maps paths
// "/plinth-dev/<repo>/tar.gz/refs/tags/<ref>" to a tarball built from the
// supplied entries. Each file is prefixed with "<repo>-<ref>/" inside the
// archive, matching GitHub's codeload format.
func starterServer(t *testing.T, repos map[string]map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /plinth-dev/<repo>/tar.gz/refs/tags/<ref>
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if len(parts) != 6 || parts[0] != "plinth-dev" || parts[2] != "tar.gz" {
			http.NotFound(w, r)
			return
		}
		repo, ref := parts[1], parts[5]
		entries, ok := repos[repo]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(buildTarball(t, repo, ref, entries))
	}))
}

func buildTarball(t *testing.T, repo, ref string, entries map[string]string) []byte {
	t.Helper()
	prefix := repo + "-" + strings.TrimPrefix(ref, "v") + "/"
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: prefix, Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
		t.Fatal(err)
	}
	for name, body := range entries {
		hdr := &tar.Header{Name: prefix + name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func mustRead(t *testing.T, root, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(body)
}
