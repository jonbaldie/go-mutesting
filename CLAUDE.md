# go-mutesting

Mutation testing for Go. Applies code mutations and checks whether tests catch them.

## Build & test

```bash
go build ./cmd/go-mutesting
go test ./...
```

**Skip these — broken by design, don't fix:**
- `internal/importing` — assumes GOPATH layout, fails in module mode
- `internal/parser` — uses deprecated `go/loader`

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

`.github/workflows/mutation.yml` runs go-mutesting on itself. Gates: MSI ≥ 40%, covered-MSI ≥ 60%.

## Conventions

- Quality gates exit with code 4 (not 1) so CI can distinguish "escaped mutants" from "tool error".
- `--min-msi` / `--min-covered-msi` CLI flags default to `-1` (sentinel for "use config or skip gate"); config zero value means no gate.
- `HasCoverage bool` on `Report` distinguishes "coverage ran, nothing uncovered" from "coverage never ran".
