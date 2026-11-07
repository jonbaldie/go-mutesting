---
name: verify-per-test-package-identity
description: Verify that recursive per-test profiling keeps each covering package's test eligible for a mutant. Use when changing per-test package discovery, temporary binary paths, profile collection, or mutant test selection.
---

# Verify per-test package identity

**Property:** For the checked recursive runs, each test that covers a mutant's line remains eligible for that mutant, or profiling falls back to the full test suite.

**Harm:** If a covering test disappears from a valid-looking map, a mutant the full suite kills can be reported as escaped.

## Coverage and sync

Last matched to commit `ba3d0680f4bacecba3bf1bfda4668ba30bc1c7cc`:

| Code | What to compare to the checks |
| --- | --- |
| `internal/coverage/coverage.go`: `ListTestPackages`, `BuildPerTestProfileForPackages`, `compileCoverageTestBinary`, `profileTests`, `runPerTestWorker`, `mergePerTestProfiles`, `parsePerTestProfile`, `applyPerTestResult`, `PerTestProfile.CoveringTests` and `CoveringTestsRelative` | Which packages and tests are profiled, how each binary and profile is named, whether jobs can observe a different package's binary, and whether incomplete profiles are rejected. |
| `internal/engine/engine.go`: `perTestForPackage`, `buildPerTestCoverageProfile`, `perTestUnavailable`, `runGoTest`, `perTestRunFilter` | The full-suite fallback and the filter passed to the mutant's `go test` invocation. |
| `internal/engine/gotest.go`: `goTestInvocation`, `perTestToolchain` | Compilation and individual-test commands and their package, binary, test name, and coverage profile arguments. |
| `internal/engine/runner.go`: `resolveRunner`; `internal/models/options.go`: `PerTest` | When the built-in runner enables per-test selection. |

From the repo root, inspect covered changes since that commit, including the working tree:

```sh
git diff ba3d0680f4bacecba3bf1bfda4668ba30bc1c7cc -- internal/coverage/coverage.go internal/engine/engine.go internal/engine/gotest.go internal/engine/runner.go internal/models/options.go
```

If the commit is unavailable, inspect all covered code. Read changed functions and compare them with `probe_test.go`, `fallback_probe_test.go`, and the CLI fixture in `check.sh`, including the order of compilation and test execution. If the implementation changes, update the checks to exercise its actual behavior; don't preserve an obsolete naming rule. Advance the sync commit only after confirming the checks still describe the code at the new commit.

## Run and interpret

From the repo root:

```sh
./.agents/skills/verify-per-test-package-identity/check.sh
```

The check needs Bash, Python 3, `sh`, Git, and the Go toolchain required by `go.mod` (last run with Go 1.26.6 on macOS). Go dependencies must already be available or downloadable. It uses temporary files and Go test overlays; it does not edit production files. Exit 0 means all checks passed, 1 reports a property violation, and 2 reports a setup or unexpected-result failure. When a prior probe found a violation but the CLI comparison cannot complete, the command reports both facts and exits 1.

The first check exhausts all 240 ordered pairs of distinct package paths formed by two segments from `a`, `b`, `a_a`, and `a_b`. It calls the real `BuildPerTestProfileForPackages` with one worker. Its controlled toolchain records a package in each compiled-binary path and writes a valid profile: both tests cover the same line if they run against their own package's binary. A complete map must include both names; an error or nil map is safe because the engine uses the full suite. The toolchain first checks its own commands, and the real coverage code also checks a disjoint-package pair.

The second check calls the real engine's fallback and run-filter functions. An unavailable or empty profile must not restrict the test suite. Compare their callers too when the code changes; the probe does not follow every caller automatically.

The third check builds the actual CLI in a temporary module. `a/b_c` has a test that kills a return mutant, and `a_b/c` has an observer that covers the same line. The full recursive suite must kill the mutant; the recursive `--per-test` run must kill it too. This checks the controlled toolchain's crucial assumption against real compiled Go test binaries.

At the sync commit, the generated cases and CLI find [upstream issue #365](https://github.com/quality-gates/mutago/issues/365). The paths `a/b_c` and `a_b/c` compile to the same temporary binary path. The second compilation replaces the first, then a valid empty profile hides the missing killer. The CLI reports the mutant escaped even though the full suite kills it. Keep this as a finding until the real code changes; do not weaken the property to make the check pass.

For any failure, confirm the control ran and replay the affected package paths through the current code. A probe setup error or an outdated controlled toolchain is not evidence of a product bug. To check sensitivity, use a disposable worktree: first give each package a distinct binary path and confirm all checks pass; then change the real path calculation there so all packages share one binary path and confirm the generated cases and CLI fail. Restore distinct paths, change `perTestRunFilter` so an absent profile returns a nonempty filter, and confirm the fallback probe fails. Discard the scratch changes and rerun against the original code.

## Boundaries

This is bounded exhaustive input checking against real Go code, not a proof over all import paths or all package sets. No direct Go verifier is configured for this unannotated code. The generated cases use exactly two packages, one distinct test name per package, one covered line, and one worker; the CLI case uses three real packages and one mutant. The controlled commands model binary ownership and valid profiles, not the Go compiler. The checks do not cover every test flag, operating system, concurrent worker schedule, duplicated test name, or differences between a test's standalone coverage and its behavior in the full suite. A pass says only that the checked cases preserved eligibility or safely fell back.
