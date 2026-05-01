# Plinth — CLI

> **Status: not yet released — Phase E in progress.**
> The Homebrew tap and `go install` paths below are the **target** install flow. Neither resolves yet — there's no Go code in this repo and no `plinth-dev/homebrew-tap` repo. Track progress on the [roadmap](https://github.com/plinth-dev/.github/blob/main/ROADMAP.md).

`plinth` will be a single-binary Go CLI that scaffolds new modules from the Plinth starters. Target: five minutes from idea to deployed-in-dev.

## Install (target — Phase E)

```bash
# Homebrew tap
brew install plinth-dev/tap/plinth

# Or via Go
go install github.com/plinth-dev/cli@latest
```

## Usage (target)

```bash
# Scaffold a module with both web and API
plinth new my-module --web --api --owner=platform-team --data-class=internal

# Web only
plinth new my-module --web

# API only
plinth new my-module --api

# Verify local toolchain
plinth doctor

# Print version
plinth version
```

## What `plinth new` does

1. Clones [`starter-web`](https://github.com/plinth-dev/starter-web) and/or [`starter-api`](https://github.com/plinth-dev/starter-api) into `my-module/` and `my-module-api/`.
2. Renames everything: module name, env var prefixes, package names, container names, Cerbos resource kind.
3. Optionally creates a GitLab project (with `--gitlab-push`) and pushes.
4. Optionally opens MRs against the GitOps repo (Argo Application) and the policies repo (default Cerbos policy) with `--open-mrs`.
5. Optionally registers the module in Backstage with `--register-backstage`.

Output is **deterministic for the same inputs** — CI compares generated structure against a checked-in golden tree on every change.

## Why both a CLI and a Backstage template

The [`scaffolder`](https://github.com/plinth-dev/scaffolder) Backstage template is the in-portal flow for app teams who already use Backstage. The CLI is the offline / scripting / first-time-clusters flow. Both produce identical output for the same inputs (CI verifies).

## Related

- [`scaffolder`](https://github.com/plinth-dev/scaffolder) — the Backstage software template.
- [`starter-web`](https://github.com/plinth-dev/starter-web) / [`starter-api`](https://github.com/plinth-dev/starter-api) — what gets cloned.
- [`plinth.run/start/try-it`](https://plinth.run/start/try-it/) — the 60-minute end-to-end tutorial.

## License

MIT — see [LICENSE](./LICENSE).
