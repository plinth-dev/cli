// Package fetch downloads and extracts starter tarballs from GitHub.
//
// GitHub serves a tarball of any tag at
// https://codeload.github.com/<owner>/<repo>/tar.gz/refs/tags/<ref>
// whose entries are prefixed with <repo>-<ref-without-v>/. Extract strips
// that prefix.
package fetch

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultBaseURL is GitHub's tarball host. Override in tests.
const DefaultBaseURL = "https://codeload.github.com"

// Client downloads and extracts starter archives.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New returns a Client with sensible defaults.
func New() *Client {
	return &Client{
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// FetchAndExtract downloads <owner>/<repo>@<ref> and extracts it into dst.
// dst must not exist; it will be created. The leading "<repo>-<ref>/" path
// component from each archive entry is stripped.
func (c *Client) FetchAndExtract(ctx context.Context, owner, repo, ref, dst string) error {
	url := fmt.Sprintf("%s/%s/%s/tar.gz/refs/tags/%s", c.BaseURL, owner, repo, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("fetch: build request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch: GET %s: status %d", url, resp.StatusCode)
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("fetch: mkdir %s: %w", dst, err)
	}

	return extract(resp.Body, dst)
}

// extract reads a gzipped tar from r and writes its contents to dst, stripping
// the first path component from every entry.
func extract(r io.Reader, dst string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("fetch: gunzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("fetch: abs %s: %w", dst, err)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("fetch: tar next: %w", err)
		}

		name := stripFirstComponent(hdr.Name)
		if name == "" {
			continue // top-level dir entry
		}

		// Reject path traversal: the cleaned absolute target must stay within dst.
		target := filepath.Join(dstAbs, name)
		if !strings.HasPrefix(target+string(filepath.Separator), dstAbs+string(filepath.Separator)) && target != dstAbs {
			return fmt.Errorf("fetch: refusing to extract outside dst: %q", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("fetch: mkdir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("fetch: mkdir %s: %w", filepath.Dir(target), err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return fmt.Errorf("fetch: create %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("fetch: write %s: %w", target, err)
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("fetch: close %s: %w", target, err)
			}
		case tar.TypeSymlink:
			// Skip symlinks — starter trees don't use them and they're a security risk.
			continue
		default:
			// Skip other entry types (hardlinks, devices, etc.).
			continue
		}
	}
	return nil
}

func stripFirstComponent(name string) string {
	name = filepath.ToSlash(name)
	idx := strings.IndexByte(name, '/')
	if idx < 0 {
		return ""
	}
	return name[idx+1:]
}
