# Plinth — CLI

`plinth` is a single-binary Go CLI that scaffolds new modules from the [`starter-web`](https://github.com/plinth-dev/starter-web) and [`starter-api`](https://github.com/plinth-dev/starter-api) starters. It downloads a pinned starter tag, rewrites identifiers to your chosen names, and (optionally) initialises a fresh git repository.

## Install

```bash
go install github.com/plinth-dev/cli/cmd/plinth@latest
```

That writes a `plinth` binary into `$(go env GOPATH)/bin`. Make sure that directory is on your `PATH`.

The Homebrew tap (`brew install plinth-dev/tap/plinth`) is still on the roadmap.

## Usage

```bash
# Scaffold a new module with both web and API tiers.
plinth new billing --module-path github.com/acme/billing-api

# Web only
plinth new billing --web

# API only
plinth new billing --api --module-path github.com/acme/billing-api

# Pin to a specific starter tag
plinth new billing --ref v0.1.0

# Verify your local toolchain
plinth doctor

# Print version
plinth version
```

`plinth new <name>` produces:

| Tier  | Directory          | Default identifier            |
|-------|--------------------|-------------------------------|
| API   | `<name>-api/`      | Go module: `github.com/example/<name>-api` |
| Web   | `<name>-web/`      | npm `name`: `<name>-web`      |

The Go module path can be overridden with `--module-path`. Output directory is the current working directory by default; use `--dir <path>` to target somewhere else.

### Flags

| Flag             | Default                              | Notes |
|------------------|--------------------------------------|-------|
| `--web`          | (off — implies both with `--api` off) | scaffold the Next.js starter |
| `--api`          | (off — implies both with `--web` off) | scaffold the Go starter |
| `--dir DIR`      | `.`                                  | parent directory for the new scaffolds |
| `--module-path PATH` | `github.com/example/<name>-api`  | Go module path for the API scaffold |
| `--ref REF`      | `v0.1.0`                             | starter tag to fetch from GitHub |
| `--no-git`       | off                                  | skip `git init` inside generated dirs |

If neither `--web` nor `--api` is passed, both are scaffolded.

### What `plinth new` does

1. Fetches `https://codeload.github.com/plinth-dev/<starter>/tar.gz/refs/tags/<ref>` over HTTPS and extracts it.
2. Rewrites a small fixed set of identifier tokens:
   - `github.com/plinth-dev/starter-api` → your `--module-path`
   - bare `starter-api` (in `cmd/server/main.go`, `docker-compose.yml`, README) → `<name>-api`
   - bare `starter-web` (in `package.json`, `src/lib/env.ts`, `instrumentation-client.ts`) → `<name>-web`
3. Skips binary files, lockfiles (`go.sum`, `pnpm-lock.yaml`), `node_modules/`, `.next/`, `dist/`, `vendor/`.
4. Runs `git init -q -b main` in each scaffolded directory unless `--no-git` is given.

It does **not** rename the sample `Items` resource (Cerbos policies, handlers, repository) — that's documented as a manual step in each starter's README so you can choose your own resource shape.

### `plinth doctor`

Reports `OK` / `FAIL` / `SKIP` for each tool the starters need:

| Tool   | Required | Minimum |
|--------|----------|---------|
| `go`   | ✓        | 1.25    |
| `git`  | ✓        | 2.30    |
| `node` | ✓        | 20.0    |
| `pnpm` | ✓        | 9.0     |
| `docker` | optional | —     |

Exit status is non-zero if any required tool is missing or below its minimum.

## Build from source

```bash
git clone https://github.com/plinth-dev/cli && cd cli
make build       # writes ./bin/plinth with version baked in via -ldflags
make test        # go test -race -cover ./...
make install     # go install with ldflags into $GOPATH/bin
```

## Roadmap

These were in scope of the original CLI vision but are deferred:

- Homebrew tap (`plinth-dev/homebrew-tap`).
- `--gitlab-push`, `--open-mrs`, `--register-backstage` integrations — these are deployment-platform-specific. Plinth's stance is that the CLI emits clean code; wiring it to your GitOps / portal of choice is a thin layer you control.
- Resource-rename helper (Items → your-thing).
- Golden-tree CI verification of generated output.

Track progress on the [main project roadmap](https://github.com/plinth-dev/.github/blob/main/ROADMAP.md).

## Related

- [`starter-web`](https://github.com/plinth-dev/starter-web) / [`starter-api`](https://github.com/plinth-dev/starter-api) — what gets cloned.
- [`scaffolder`](https://github.com/plinth-dev/scaffolder) — Backstage software template (parallel scaffolder for portal users).

## License

MIT — see [LICENSE](./LICENSE).
