# Changelog

All notable changes to this project will be documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/). This project uses [Semantic Versioning](https://semver.org/).

---

## [v2.6.7] — 2026-05-19

### Added
- `conditional/bool-literal` mutator: swaps `true`↔`false` in assignments and function call arguments.
- `conditional/not` mutator: removes the `!` operator from negated conditions in `if`, `for`, and `&&`/`||` operands.
- `expression/string-literal` mutator: replaces non-empty string literals in `==`/`!=` comparisons with `""`.
- `statement/defer-remove` mutator: turns `defer f()` into an immediate call, testing whether deferred execution timing matters. Covers `defer` inside `select` case branches.
- `arithmetic/assign_invert` expanded to cover bitwise compound assignments (`&=`, `|=`, `^=`, `<<=`, `>>=`, `&^=`).
- `--logger-gitlab` flag: emits a GitLab Code Quality artifact JSON (`go-mutesting-gitlab.json`).
- `--timeout-coefficient` flag: scales per-mutation timeout by a multiplier of the baseline test-suite run time.
- `--run-mutant-id` flag: runs only the mutant with a given stable ID (copy the `id` field from `go-mutesting-agentic.json`).
- `ignore_source_lines` config key: list of regexes; mutations on matching source lines are suppressed.
- `SourceLineRegexFilter` in `internal/filter`: programmatic filter for skipping mutations on regex-matched lines.
- New tests for `internal/coverage` (`CountTests`, `BuildPerTestProfile`) and `internal/filter` (`SourceLineRegexFilter`, `ShouldSkip`), raising quality-gate MSI to 77.75% / 83.78% covered.

### Fixed
- `statement/return` mutator now zeroes struct-typed return values using an empty composite literal (`T{}`).
- Guard conditions in `expression/context-nil`, `expression/string-literal`, and `statement/remove` refactored from `==` string comparisons to `switch`, eliminating equivalent mutations.
- GitLab report fingerprint now uses the stable `baseline.MutantID` hash, preventing deduplication of distinct mutations on the same line.

---

## [v2.6.6] — 2026-05-19

### Added
- `--dry-run` now prints per-mutator counts as a summary table before the grand total.
- `--per-test` startup message: prints the package name and test count before building the per-test coverage map.
- Agentic JSON `context_start_line` field: anchors `context_lines[0]` to a 1-based source line so LLMs can navigate without guessing.

### Fixed
- `--git-diff-base` now auto-detects the default branch via `git symbolic-ref origin/HEAD`; falls back to `master`.
- Agentic JSON `description` for simple one-line mutations now shows the exact change (e.g. `` `return a, b` → `return a, nil` ``).
- `--quiet` help text now mentions `--no-diffs`.

[v2.6.6]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.6.6

---

## [v2.6.5] — 2026-05-19

### Fixed
- `silent_mode: true` now prints only the final summary line; previously it suppressed the summary as well.
- `--logger-agentic-json` descriptions and kill hints were missing for 14 mutators; all 27 current mutators are now covered.
- `statement/remove` no longer generates false escapes on blank-assign statements (`_, _ = a, b`).
- `.gitignore` extended to cover `go-mutesting-report.html`, `go-mutesting-summary.json`, and `go-mutesting-agentic.json`.

[v2.6.5]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.6.5

---

## [v2.6.4] — 2026-05-19

### Added
- `disable_mutators` and `enable_mutators` config keys: commit per-mutator control to YAML instead of threading flags through every CLI call. Trailing-`*` wildcards work for whole categories (`arithmetic/*`).
- Config JSON Schema updated for editor autocomplete.

### Fixed
- Panic when `*` was passed as a bare wildcard pattern to `--disable`.

[v2.6.4]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.6.4

---

## [v2.6.3] — 2026-05-19

### Added
- `--dry-run` flag: count mutations without writing files or running tests.
- `--no-diffs` flag: suppress diff output for all results (good for CI pipelines that consume the JSON report).
- `--output-statuses` flag: filter terminal output to specific result types (e.g. `e` for escaped, `ke` for killed + escaped).
- Config JSON Schema at `schema/config-schema.json`; add a comment to your config file for editor validation and autocomplete.

### Fixed
- Diffs for escaped and errored mutants now respect `--output-statuses`; previously they leaked through when other statuses were suppressed.

[v2.6.3]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.6.3

---

## [v2.6.2] — 2026-05-18

### Added
- `--per-test` flag: builds a per-test coverage map and runs only the tests that cover each mutation.
- `--test-flags` flag: passes extra flags to every `go test` call (e.g. `--test-flags=-short`).

### Fixed
- `--per-test` worker used `return` instead of `continue` on parse errors, causing a deadlock with one worker when any test binary failed to compile.
- `--test-flags` values were not forwarded to the per-test profile-building phase, causing inconsistent behaviour.
- `--per-test` help text incorrectly stated it requires `--coverage`.

[v2.6.2]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.6.2

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

[v2.6.1]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.6.1

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

[v2.6.7]: https://github.com/jonbaldie/go-mutesting/releases/tag/v2.6.7
[Unreleased]: https://github.com/jonbaldie/go-mutesting/compare/v2.6.7...HEAD
