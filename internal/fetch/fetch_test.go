package fetch

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
)

func TestFetchAndExtract(t *testing.T) {
	body := buildTarball(t, map[string]string{
		"starter-api-0.1.0/":                   "",
		"starter-api-0.1.0/go.mod":             "module github.com/plinth-dev/starter-api\n\ngo 1.25.0\n",
		"starter-api-0.1.0/cmd/server/main.go": "package main\n",
		"starter-api-0.1.0/README.md":          "# starter-api\n",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/plinth-dev/starter-api/tar.gz/refs/tags/v0.1.0"; r.URL.Path != want {
			t.Errorf("unexpected path %q, want %q", r.URL.Path, want)
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(body)
	}))
	defer srv.Close()

	dst := t.TempDir()
	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	if err := c.FetchAndExtract(context.Background(), "plinth-dev", "starter-api", "v0.1.0", dst); err != nil {
		t.Fatalf("FetchAndExtract: %v", err)
	}

	gomod, err := os.ReadFile(filepath.Join(dst, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if !strings.Contains(string(gomod), "module github.com/plinth-dev/starter-api") {
		t.Errorf("go.mod missing module line: %q", gomod)
	}

	if _, err := os.Stat(filepath.Join(dst, "cmd", "server", "main.go")); err != nil {
		t.Errorf("missing cmd/server/main.go: %v", err)
	}
}

func TestExtractRejectsTraversal(t *testing.T) {
	body := buildTarball(t, map[string]string{
		"starter-api-0.1.0/":               "",
		"starter-api-0.1.0/../escape.txt": "nope\n",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	dst := t.TempDir()
	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	err := c.FetchAndExtract(context.Background(), "plinth-dev", "starter-api", "v0.1.0", dst)
	if err == nil || !strings.Contains(err.Error(), "outside dst") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func TestFetchAndExtractNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dst := t.TempDir()
	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	err := c.FetchAndExtract(context.Background(), "plinth-dev", "starter-api", "v9.9.9", dst)
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected 404 error, got %v", err)
	}
}

func TestStripFirstComponent(t *testing.T) {
	cases := map[string]string{
		"starter-api-0.1.0/":            "",
		"starter-api-0.1.0/go.mod":      "go.mod",
		"starter-api-0.1.0/a/b/c.txt":   "a/b/c.txt",
		"justonecomponent":              "",
	}
	for in, want := range cases {
		if got := stripFirstComponent(in); got != want {
			t.Errorf("stripFirstComponent(%q) = %q, want %q", in, got, want)
		}
	}
}

// buildTarball returns a gzipped tar where each map entry becomes a file
// (or directory if its name ends in "/").
func buildTarball(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for name, body := range entries {
		hdr := &tar.Header{Name: name, Mode: 0o644}
		if strings.HasSuffix(name, "/") {
			hdr.Typeflag = tar.TypeDir
			hdr.Mode = 0o755
		} else {
			hdr.Typeflag = tar.TypeReg
			hdr.Size = int64(len(body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("WriteHeader: %v", err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(body)); err != nil {
				t.Fatalf("tar write: %v", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tw.Close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gz.Close: %v", err)
	}
	return buf.Bytes()
}
