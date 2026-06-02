# Quick Start

## 1. Install

```bash
go install github.com/jonbaldie/go-mutesting/v2/cmd/go-mutesting@latest
```

## 2. Run against your package

```bash
go-mutesting ./...
```

Each mutation prints `KILLED` (tests detected it) or `ESCAPED` (tests missed it). Escaped mutants print a diff showing exactly what changed.

## 3. Add coverage-awareness

Without `--coverage`, uncovered code inflates the "not covered" bucket and can produce a misleadingly low MSI. With it, mutants on untested lines are separated out and excluded from the covered-MSI denominator.

```bash
go-mutesting --coverage ./...
```

## 4. Set quality gates

```bash
go-mutesting --coverage --min-msi 70 --min-covered-msi 80 ./...
```

Exit code 4 means a gate wasn't met. Exit code 0 means all gates passed.

## 5. Reduce noise

Use `--quiet` to suppress killed/skipped output and only show escaped mutants and the summary.

```bash
go-mutesting --quiet --coverage ./...
```

## 6. Limit to changed lines (PR mode)

```bash
go-mutesting \
  --git-diff-lines \
  --git-diff-base main \
  --ignore-msi-with-no-mutations \
  --min-msi 80 \
  ./...
```

## 7. Get LLM-ready suggestions

```bash
go-mutesting --logger-agentic-json --quiet ./...
# Feed go-mutesting-agentic.json to an LLM for targeted test suggestions.
```
