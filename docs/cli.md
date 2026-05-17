# CLI Reference

Run `go-mutesting --help` for the authoritative flag list. The most commonly used flags are documented here.

## Targets

```
go-mutesting [flags] <pkg|file|dir> ...
```

Targets can be Go source files, directories, or import paths. The `...` wildcard searches recursively. Test files (`_test.go`) are excluded automatically.

## Core flags

| Flag | Default | Description |
| :--- | :------ | :---------- |
| `--exec` | (built-in) | Custom exec command for testing each mutation |
| `--exec-timeout` | `10` | Seconds to wait before killing the test process |
| `--workers` | all CPUs | Number of parallel mutation workers |
| `--config` | — | Path to YAML config file |

## Output flags

| Flag | Description |
| :--- | :---------- |
| `--quiet` | Suppress killed/skipped lines; show only escaped mutants and summary |
| `--verbose` | Print full test output for each mutation |
| `--debug` | Print internal debug information |
| `--logger-github` | Emit escaped mutants as `::warning` GitHub Actions annotations |
| `--logger-summary-json` | Write compact stats to `go-mutesting-summary.json` |
| `--logger-agentic-json` | Write LLM-ready report to `go-mutesting-agentic.json` |

## Quality gates

| Flag | Default | Description |
| :--- | :------ | :---------- |
| `--min-msi` | `-1` (disabled) | Fail (exit 4) if overall MSI is below this value (0–100) |
| `--min-covered-msi` | `-1` (disabled) | Fail (exit 4) if covered-code MSI is below this value |
| `--fail-on-escaped` | `false` | Fail (exit 4) if any mutant survives, without requiring `--min-msi` |
| `--ignore-msi-with-no-mutations` | `false` | Exit 0 when no mutations are generated (useful in PR mode) |

## Coverage

| Flag | Description |
| :--- | :---------- |
| `--coverage` | Run `go test -coverprofile` first; exclude uncovered lines from covered-MSI |

## Filtering

| Flag | Description |
| :--- | :---------- |
| `--blacklist <file>` | File of MD5 checksums to skip |
| `--disable <mutator>` | Disable a mutator by name (repeatable) |
| `--git-diff-lines` | Only mutate lines changed since `--git-diff-base` |
| `--git-diff-base` | Git ref to diff against (default: `HEAD`) |

## Baseline

| Flag | Description |
| :--- | :---------- |
| `--baseline <file>` | Known-survivors file; only fail on new escapes |
| `--update-baseline` | Record current survivors and exit 0 |

## Exit codes

| Code | Meaning |
| :--- | :------ |
| 0 | All mutations tested; all quality gates passed |
| 1 | Internal error |
| 4 | A quality gate was not met (`--min-msi`, `--min-covered-msi`, or `--fail-on-escaped`) |
