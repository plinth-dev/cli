package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestDoctor_AllPresent(t *testing.T) {
	env := doctorEnv{
		lookPath: func(string) (string, error) { return "/usr/bin/x", nil },
		output: func(name string, _ ...string) ([]byte, error) {
			switch name {
			case "go":
				return []byte("go version go1.25.0 darwin/arm64"), nil
			case "git":
				return []byte("git version 2.45.0"), nil
			case "node":
				return []byte("v22.0.0"), nil
			case "pnpm":
				return []byte("9.10.0"), nil
			case "docker":
				return []byte("Docker version 27.0.0, build abcd1234"), nil
			}
			return nil, errors.New("unknown")
		},
	}
	var stdout, stderr bytes.Buffer
	code := runDoctorWithEnv(nil, &stdout, &stderr, env)
	if code != 0 {
		t.Fatalf("exit=%d stdout=%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "All required tools available.") {
		t.Errorf("missing OK summary: %s", stdout.String())
	}
}

func TestDoctor_MissingRequired(t *testing.T) {
	env := doctorEnv{
		lookPath: func(name string) (string, error) {
			if name == "node" {
				return "", errors.New("not found")
			}
			return "/usr/bin/x", nil
		},
		output: func(name string, _ ...string) ([]byte, error) {
			switch name {
			case "go":
				return []byte("go version go1.25.0"), nil
			case "git":
				return []byte("git version 2.45.0"), nil
			case "pnpm":
				return []byte("9.10.0"), nil
			case "docker":
				return []byte("Docker version 27.0.0"), nil
			}
			return nil, errors.New("unknown")
		},
	}
	var stdout, stderr bytes.Buffer
	code := runDoctorWithEnv(nil, &stdout, &stderr, env)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d, stdout=%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "node       FAIL") {
		t.Errorf("expected node FAIL in stdout: %s", stdout.String())
	}
}

func TestDoctor_BelowMinVersion(t *testing.T) {
	env := doctorEnv{
		lookPath: func(string) (string, error) { return "/usr/bin/x", nil },
		output: func(name string, _ ...string) ([]byte, error) {
			switch name {
			case "go":
				return []byte("go version go1.23.0 darwin/arm64"), nil
			case "git":
				return []byte("git version 2.45.0"), nil
			case "node":
				return []byte("v22.0.0"), nil
			case "pnpm":
				return []byte("9.10.0"), nil
			case "docker":
				return []byte("Docker version 27.0.0"), nil
			}
			return nil, errors.New("unknown")
		},
	}
	var stdout, stderr bytes.Buffer
	code := runDoctorWithEnv(nil, &stdout, &stderr, env)
	if code != 1 {
		t.Fatalf("expected exit 1 for old go, got %d", code)
	}
	if !strings.Contains(stdout.String(), "need >= 1.25") {
		t.Errorf("expected min-version detail: %s", stdout.String())
	}
}

func TestDoctor_OptionalDockerSkip(t *testing.T) {
	env := doctorEnv{
		lookPath: func(name string) (string, error) {
			if name == "docker" {
				return "", errors.New("not found")
			}
			return "/usr/bin/x", nil
		},
		output: func(name string, _ ...string) ([]byte, error) {
			switch name {
			case "go":
				return []byte("go version go1.25.0"), nil
			case "git":
				return []byte("git version 2.45.0"), nil
			case "node":
				return []byte("v22.0.0"), nil
			case "pnpm":
				return []byte("9.10.0"), nil
			}
			return nil, errors.New("unknown")
		},
	}
	var stdout, stderr bytes.Buffer
	code := runDoctorWithEnv(nil, &stdout, &stderr, env)
	if code != 0 {
		t.Fatalf("expected 0 with docker skipped, got %d, stdout=%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "docker     SKIP") {
		t.Errorf("expected docker SKIP: %s", stdout.String())
	}
}
