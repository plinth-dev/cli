package rename

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyAPI(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, root, "go.mod", `module github.com/plinth-dev/starter-api

go 1.25.0

require github.com/plinth-dev/sdk-go/audit v0.1.0
`)
	mustWrite(t, root, "cmd/server/main.go", `package main

import (
	"github.com/plinth-dev/sdk-go/audit"
	"github.com/plinth-dev/starter-api/internal/handlers"
)

// starter-api entry point
var _ = handlers.X
var _ = audit.Event{}
`)
	mustWrite(t, root, "docker-compose.yml", "services:\n  api:\n    environment:\n      SERVICE_NAME: starter-api\n")
	mustWrite(t, root, "go.sum", "github.com/plinth-dev/starter-api should-not-be-touched\n")
	// node_modules content must be skipped wholesale.
	mustWrite(t, root, "node_modules/some-pkg/index.js", "starter-api should-not-be-touched\n")

	if err := Apply(root, ForAPI("github.com/acme/billing-api", "billing-api")); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	gomod := mustRead(t, root, "go.mod")
	if !strings.Contains(gomod, "module github.com/acme/billing-api") {
		t.Errorf("go.mod module path not rewritten: %s", gomod)
	}
	if !strings.Contains(gomod, "github.com/plinth-dev/sdk-go/audit") {
		t.Errorf("sdk-go dep was corrupted: %s", gomod)
	}

	mainGo := mustRead(t, root, "cmd/server/main.go")
	if !strings.Contains(mainGo, `"github.com/acme/billing-api/internal/handlers"`) {
		t.Errorf("internal import not rewritten: %s", mainGo)
	}
	if !strings.Contains(mainGo, `"github.com/plinth-dev/sdk-go/audit"`) {
		t.Errorf("sdk-go import was corrupted: %s", mainGo)
	}
	if !strings.Contains(mainGo, "// billing-api entry point") {
		t.Errorf("comment service-name not rewritten: %s", mainGo)
	}

	dc := mustRead(t, root, "docker-compose.yml")
	if !strings.Contains(dc, "SERVICE_NAME: billing-api") {
		t.Errorf("docker-compose service name not rewritten: %s", dc)
	}

	if got := mustRead(t, root, "go.sum"); !strings.Contains(got, "starter-api should-not-be-touched") {
		t.Errorf("go.sum was rewritten (must be skipped): %s", got)
	}
	if got := mustRead(t, root, "node_modules/some-pkg/index.js"); !strings.Contains(got, "starter-api should-not-be-touched") {
		t.Errorf("node_modules was rewritten (must be skipped): %s", got)
	}
}

func TestApplyWeb(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, root, "package.json", `{"name": "starter-web", "version": "0.1.0"}`)
	mustWrite(t, root, "src/lib/env.ts", `SERVICE_NAME: z.string().default("starter-web")`)
	mustWrite(t, root, "instrumentation-client.ts", `serviceName: process.env.NEXT_PUBLIC_SERVICE_NAME ?? "starter-web"`)

	if err := Apply(root, ForWeb("billing-web")); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	pkg := mustRead(t, root, "package.json")
	if !strings.Contains(pkg, `"name": "billing-web"`) {
		t.Errorf("package.json name not rewritten: %s", pkg)
	}
	env := mustRead(t, root, "src/lib/env.ts")
	if !strings.Contains(env, `default("billing-web")`) {
		t.Errorf("env.ts default not rewritten: %s", env)
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, root, "go.mod", "module github.com/plinth-dev/starter-api\n")

	repls := ForAPI("github.com/acme/billing-api", "billing-api")
	if err := Apply(root, repls); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	want := mustRead(t, root, "go.mod")
	if err := Apply(root, repls); err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if got := mustRead(t, root, "go.mod"); got != want {
		t.Errorf("second Apply changed file: %q vs %q", got, want)
	}
}

func TestApplySkipsBinaries(t *testing.T) {
	root := t.TempDir()
	binary := append([]byte("starter-web\x00binary"), 0x01, 0x02, 0x03)
	mustWriteRaw(t, root, "logo.png", binary)

	if err := Apply(root, ForWeb("billing-web")); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(binary) {
		t.Errorf("binary file rewritten: %v", got)
	}
}

func mustWrite(t *testing.T, root, rel, body string) {
	t.Helper()
	mustWriteRaw(t, root, rel, []byte(body))
}

func mustWriteRaw(t *testing.T, root, rel string, body []byte) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, root, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
