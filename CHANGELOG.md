# Changelog

All notable changes to this project will be documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/). This project uses [Semantic Versioning](https://semver.org/).

---

## [v2.6.1] — 2026-05-18

### Added
- `numbers/float-negate` mutator: replaces non-zero float literals with `0.0`.
- `arithmetic/negate` mutator: inverts unary negation (`-x → +x`), closing the gap with gremlins' INVERT_NEGATIVES.
- `statement/remove-self-assign` mutator: removes self-assignment statements (`a = a`), closing the gap with gremlins' REMOVE_SELF_ASSIGNMENTS.
- `expression/context-nil` mutator: replaces `context.Context` arguments at call sites with `nil`.
- `expression/error-guard` mutator: replaces `if err != nil` with `if false` and `if err == nil` with `if true`.
- Public mutator extension API: `mutator.Register` / `mutator.New` so third-party packages can add custom operators without forking.
- MkDocs documentation site (Install, Quick Start, CLI reference, per-mutator pages, CI integration guide, JSON output schemas). Deployed to GitHub Pages.
- This CHANGELOG.

---

## [v2.6.0] — 2026-05-17

### Added
- `statement/return` mutator: replaces return values with the zero value for their type (`false`, `0`, `""`, `nil`). Uses `go/types` for type resolution. Kills 91% of its own mutants on first run.
- Copyright 2026 Jonathan Baldie to LICENSE.

### Fixed
- Progress goroutine is now joined (via `sync.WaitGroup`) before the summary line prints, eliminating a race on terminals.
- Config file parsing now rejects unknown YAML keys (`KnownFields(true)`) instead of silently ignoring them.
- `internal/parser`: migrated `ParseAndTypeCheckFile` from the deprecated `golang.org/x/tools/go/loader` to `golang.org/x/tools/go/packages`. Build-constrained files (e.g. testdata fixtures) fall back to direct parsing via `go/types.Config`.
- Vulnerability scanning via `govulncheck` added to CI (advisory-only until go1.26.3 releases).

[v2.6.0]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.6.0

---

## [v2.5.2] — 2026-05-17

### Fixed
- Updated module import paths to use `/v2` suffix consistently.
- `govulncheck` step made advisory-only in CI (`|| true`) to avoid blocking on unfixable CVEs in go1.25.x.
- Five bughunting fixes: exec path splitting (`strings.Fields`), covered-MSI no-coverage sentinel, JSON report conditional, `--quiet` suppressing NOT COVERED lines, progress goroutine race.

[v2.5.2]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.5.2

---

## [v2.5.1] — 2026-05-17

### Fixed
- CI workflow package paths updated to use `/v2` module suffix.

[v2.5.1]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.5.1

---

## [v2.5.0] — 2026-05-17

### Added
- Live progress display on terminals (shows running kill/escape/skip counts at 200 ms intervals; suppressed in verbose/debug/silent mode).
- `--baseline` / `--update-baseline` flags: track known-surviving mutants in a file; CI only fails on *new* regressions.
- `--logger-agentic-json`: writes `go-mutesting-agentic.json` with enriched survived-mutant data designed for LLM consumption.

[v2.5.0]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.5.0

---

## [v2.4.3] — 2026-05-17

### Added
- Parallel mutation execution via `go test -overlay` (all CPUs by default, configurable with `--workers`).
- `--noop` pre-flight check: runs the test suite once without mutations; exits with an error if it already fails.
- `--logger-summary-json`: writes compact stats JSON to `go-mutesting-summary.json`.
- `select/case-remove` and `select/default-remove` mutators (Go-specific channel select mutations).
- `concurrency/goroutine-remove` mutator (removes the `go` keyword from goroutine calls).

### Fixed
- `saveAST` errors now tracked in stats instead of silently dropped.
- Git diff line calculation uses the first hunk line for multi-hunk diffs.

[v2.4.3]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.4.3

---

## [v2.4.2] — 2026-05-17

### Fixed
- Internal test and documentation fixes; no behaviour change.

[v2.4.2]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.4.2

---

## [v2.4.1] — 2026-05-17

### Added
- `conditional/negated` mutator.
- `--quiet` flag: suppresses killed/skipped output, shows only escaped mutants and the final summary.
- `--fail-on-escaped`: exits with code 4 if any mutant survives, without requiring `--min-msi`.
- Vulnerability scan via `govulncheck` wired into CI.

[v2.4.1]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.4.1

---

## [v2.4.0] — 2026-05-17

### Changed
- Module path renamed from `github.com/avito-tech/go-mutesting` to `github.com/jonbaldie/go-mutesting/v2`.

[v2.4.0]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.4.0

---

[Unreleased]: https://github.com/jonbaldie/go-mutesting/compare/v2.6.0...HEAD
