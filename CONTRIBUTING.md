# Contributing to chaotic

Thanks for contributing! This repo is a multi-module Go workspace
(`go.work`): the root module `github.com/RomanAgaltsev/chaotic` plus submodules under
`adapter/`, `observer/`, and `source/`.

## Prerequisites

- Go 1.26+
- [Task](https://taskfile.dev): `go install github.com/go-task/task/v3/cmd/task@latest`
- golangci-lint v2: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- govulncheck: `go install golang.org/x/vuln/cmd/govulncheck@latest`

## Everyday commands

All commands run through `Taskfile.yml` so local == CI:

```bash
task              # list available tasks
task ci           # full local gate: tidy-check, vet, lint, race tests, coverage, alloc-gate, vuln
task test:all     # race tests across every module
task lint:all     # golangci-lint across every module
task fmt           # gofumpt formatting (golangci-lint fmt)
```

Per-module commands take a `MODULE` variable, e.g. `task test:race MODULE=adapter/grpc`.

## Commit & PR conventions

- We **squash-merge** PRs. The **PR title** becomes the commit on `main` and
  drives release-please, so it **must** be a
  [Conventional Commit](https://www.conventionalcommits.org/): `feat: ...`,
  `fix: ...`, `chore: ...`, `docs: ...`, `refactor: ...`, `test: ...`,
  `build: ...`, `ci: ...`, `perf: ...`. Scope optional: `feat(engine): ...`.
- Breaking changes: add `!` (`feat!: ...`) or a `BREAKING CHANGE:` footer.
  Pre-1.0 this bumps the minor version.
- A `pr-title` check enforces the convention.

## Before opening a PR

Run `task ci` and make sure it is green.

## Releasing

Releases are automated by [release-please](https://github.com/googleapis/release-please-action).
Merging Conventional-Commit PRs to `main` keeps a standing **release PR** that
updates `CHANGELOG.md` and the version. Merge that PR to cut the release:

- Root module → tag `vX.Y.Z`
- `adapter/grpc` → tag `adapter/grpc/vX.Y.Z`

### Submodule pre-release checklist (FIRST publish of any submodule)

A submodule that depends on the root module must not ship the dev-time
`replace` directive (consumers ignore it). Before merging a submodule's first
release PR:

1. Ensure the **root** module is released at `vX.Y.Z`.
2. In the submodule's `go.mod`, set `require github.com/RomanAgaltsev/chaotic vX.Y.Z`
   and **remove** the `replace github.com/RomanAgaltsev/chaotic => ../..` line.
3. Confirm local dev still resolves via `go.work` (not `replace`).
4. Run `task ci`, then merge the submodule release PR.

Until this is done, do not merge a submodule release PR.

## Maintainer setup (one-time)

These steps require repo-admin access and cannot be committed as code:

- [ ] **Codecov:** add the repo at codecov.io and set the `CODECOV_TOKEN` repo
  secret (`Settings → Secrets and variables → Actions`).
- [ ] **Renovate:** install the [Mend Renovate GitHub App](https://github.com/apps/renovate)
  on `agar/chaotic`; merge the onboarding PR.
- [ ] **Actions permissions:** `Settings → Actions → General → Workflow
      permissions` = Read and write + "Allow GitHub Actions to create and
  approve pull requests" (so release-please can open release PRs).
- [ ] **Code scanning:** ensure CodeQL/Code scanning is enabled (default once
  `security.yml` runs).
- [ ] **Branch protection:** run `bash scripts/branch-protection.sh` (requires
  `gh auth login`). Re-run if a required check name changes.