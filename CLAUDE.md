# go-mutesting

Mutation testing for Go. Applies code mutations and checks whether tests catch them.

## Build & test

```bash
go build ./cmd/go-mutesting
go test ./...
```

All packages pass. `internal/importing` and `internal/parser` were once broken in module mode but are fixed; do not skip them.

## Key packages

| Package | What it does |
| :--- | :--- |
| `cmd/go-mutesting/` | Binary entrypoint; all flag wiring and orchestration |
| `mutator/` | Mutator implementations (arithmetic, branch, expression, loop, numbers, statement) |
| `internal/models/` | `Report`, `Stats`, `Mutant` types; MSI and quality gate logic |
| `internal/gitdiff/` | Git diff line filter for `--git-diff-lines` |
| `internal/filter/` | Annotation and skip filters |
| `internal/coverage/` | Coverage profile parsing for `--coverage` |

## Self-mutation

`.github/workflows/mutation.yml` runs go-mutesting on itself. Gates: MSI ≥ 75%, covered-MSI ≥ 80%.

## Conventions

- Quality gates exit with code 4 (not 1) so CI can distinguish "escaped mutants" from "tool error".
- `--min-msi` / `--min-covered-msi` CLI flags default to `-1` (sentinel for "use config or skip gate"); config zero value means no gate.
- `HasCoverage bool` on `Report` distinguishes "coverage ran, nothing uncovered" from "coverage never ran".

## Testing posture

Integration tests live in `cmd/go-mutesting/main_test.go`. They invoke `mainCmd` directly and capture stdout+stderr.

**Do not assert on hardcoded mutation counts.** Counts change whenever a mutator is added or the example test suite improves. They are implementation details, not public behaviour.

**Assert on behaviour instead:**
- The summary line appears (`"mutation score"` is always in the output).
- Exit codes are correct (`returnOk`, `returnMsiThresholdNotMet`, etc.).
- JSON report totals are internally consistent: `TotalMutantsCount == KilledCount + EscapedCount + ErrorCount + SkippedCount + NotCoveredCount`.
- Collection lengths match stat fields: `len(Escaped) == EscapedCount`, `len(Killed) == KilledCount`.
- Each escaped mutant's `ProcessOutput` contains `"FAIL"`; each killed one contains `"PASS"`.
- MSI is in `[0.0, 1.0]`.

**For quality gate tests that must fail**, use a threshold that is permanently out of reach (e.g. `--min-msi 101`) rather than relying on the example package having escaped mutants. The example suite now kills 100% of mutants.

**After running `go test ./cmd/go-mutesting/`**, always run `git restore example/example.go`. The integration tests invoke the mutation binary against the example package and the file is sometimes left with mutations applied.
