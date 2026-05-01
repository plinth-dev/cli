// Package rename rewrites starter scaffolds in place: it replaces a fixed set
// of identifier tokens (Go module path, service name, npm package name) with
// user-chosen values across every text file under a directory.
package rename

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Replacement is an ordered string substitution applied to file bodies.
//
// Order matters: longer/more-specific tokens MUST come before shorter ones
// they share a prefix with. For example, replacing the full Go module path
// `github.com/plinth-dev/starter-api` must run before the bare service-name
// token `starter-api` so the latter does not corrupt the former.
type Replacement struct {
	Old string
	New string
}

// Apply walks root and rewrites every text file body using replacements.
// Binary files, dependency directories, and lockfiles are skipped — see
// shouldSkip / looksBinary for the policy.
func Apply(root string, replacements []Replacement) error {
	if len(replacements) == 0 {
		return nil
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipFile(d.Name()) {
			return nil
		}
		return rewriteFile(path, replacements)
	})
}

func rewriteFile(path string, replacements []Replacement) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("rename: read %s: %w", path, err)
	}
	if looksBinary(body) {
		return nil
	}

	updated := body
	for _, r := range replacements {
		if r.Old == "" || r.Old == r.New {
			continue
		}
		if !strings.Contains(string(updated), r.Old) {
			continue
		}
		updated = []byte(strings.ReplaceAll(string(updated), r.Old, r.New))
	}
	if string(updated) == string(body) {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("rename: stat %s: %w", path, err)
	}
	if err := os.WriteFile(path, updated, info.Mode().Perm()); err != nil {
		return fmt.Errorf("rename: write %s: %w", path, err)
	}
	return nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".next", "dist", "build", "vendor", "coverage", ".turbo", ".pnpm-store":
		return true
	}
	return false
}

func shouldSkipFile(name string) bool {
	switch name {
	case "go.sum", "pnpm-lock.yaml", "package-lock.json", "yarn.lock":
		return true
	}
	switch filepath.Ext(name) {
	case ".tsbuildinfo", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".svg",
		".woff", ".woff2", ".ttf", ".otf", ".eot",
		".zip", ".gz", ".tgz", ".tar", ".bz2", ".xz", ".7z",
		".pdf", ".mp3", ".mp4", ".mov", ".webm",
		".wasm":
		return true
	}
	return false
}

// looksBinary reports whether body appears to be binary content. We treat any
// embedded NUL byte in the first 8 KiB as a binary signal — matches what `git`
// does for diff/grep heuristics.
func looksBinary(body []byte) bool {
	const probe = 8 << 10
	if len(body) > probe {
		body = body[:probe]
	}
	for _, b := range body {
		if b == 0 {
			return true
		}
	}
	return false
}

// ForAPI returns the standard replacement set for the starter-api scaffold.
//
// modulePath is the full Go module path the user wants
// (e.g. "github.com/acme/billing-api"). serviceName is the bare service
// identifier baked into log lines, OTel operation names, and docker-compose
// (e.g. "billing-api"). The returned replacements are ordered: full module
// path first, then bare service name.
func ForAPI(modulePath, serviceName string) []Replacement {
	return []Replacement{
		{Old: "github.com/plinth-dev/starter-api", New: modulePath},
		{Old: "starter-api", New: serviceName},
	}
}

// ForWeb returns the replacement set for the starter-web scaffold.
// packageName is the npm package name and service identifier
// (e.g. "billing-web").
func ForWeb(packageName string) []Replacement {
	return []Replacement{
		{Old: "starter-web", New: packageName},
	}
}
