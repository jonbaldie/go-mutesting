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

## Self-mutation and quality gates

`.github/workflows/mutation.yml` runs go-mutesting on itself with two hard gates:

| Gate | Threshold | Flag |
| :--- | :--- | :--- |
| Overall MSI | ≥ 75% | `--min-msi 75` |
| Covered-code MSI | ≥ 80% | `--min-covered-msi 80` |

**Run the gates locally before committing.** Build first, then run against the same package list CI uses:

```bash
go build -o /tmp/go-mutesting ./cmd/go-mutesting
/tmp/go-mutesting \
  --exec-timeout 30 --coverage --min-msi 75 --min-covered-msi 80 \
  github.com/jonbaldie/go-mutesting/v2/mutator/arithmetic \
  github.com/jonbaldie/go-mutesting/v2/mutator/branch \
  github.com/jonbaldie/go-mutesting/v2/mutator/concurrency \
  github.com/jonbaldie/go-mutesting/v2/mutator/conditional \
  github.com/jonbaldie/go-mutesting/v2/mutator/expression \
  github.com/jonbaldie/go-mutesting/v2/mutator/loop \
  github.com/jonbaldie/go-mutesting/v2/mutator/numbers \
  github.com/jonbaldie/go-mutesting/v2/mutator/select \
  github.com/jonbaldie/go-mutesting/v2/mutator/statement \
  github.com/jonbaldie/go-mutesting/v2/internal/filter \
  github.com/jonbaldie/go-mutesting/v2/internal/coverage \
  github.com/jonbaldie/go-mutesting/v2/internal/gitdiff \
  github.com/jonbaldie/go-mutesting/v2/internal/models
```

Exit code 4 means the gate failed (escaped mutants). Exit code 0 means all gates passed.

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
