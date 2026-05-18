# JSON Output Schemas

go-mutesting can write two JSON files after each run.

## `--logger-summary-json`

Writes `go-mutesting-summary.json`. Useful for CI badges, dashboards, and downstream scripts.

```json
{
  "total": 42,
  "killed": 35,
  "escaped": 5,
  "errored": 0,
  "skipped": 2,
  "not_covered": 0,
  "msi": 0.8333,
  "covered_msi": 0.9211
}
```

| Field | Type | Description |
| :---- | :--- | :---------- |
| `total` | int | Total mutations generated |
| `killed` | int | Mutations caught by tests |
| `escaped` | int | Mutations not caught (test gaps) |
| `errored` | int | Mutations that caused a build or test error |
| `skipped` | int | Mutations skipped (blacklisted or annotated) |
| `not_covered` | int | Mutations on lines with no coverage (requires `--coverage`) |
| `msi` | float | Mutation Score Indicator: killed / total, range 0–1 |
| `covered_msi` | float | MSI restricted to covered lines only, range 0–1 |

`msi` and `covered_msi` are in the 0–1 range (not 0–100).

## `--logger-agentic-json`

Writes `go-mutesting-agentic.json`. A richer payload designed for LLM consumption. Each survived mutant gets a stable ID, the unified diff, surrounding context lines, nearby test file paths, a plain-English description of the mutation, and a hint for writing a killing test.

```json
{
  "survived_mutants": [
    {
      "id": "abc123",
      "file": "pkg/foo/foo.go",
      "line": 42,
      "mutator": "branch/if",
      "diff": "--- Original\n+++ Mutated\n...",
      "context_lines": ["func Foo() {", "  if x > 0 {", "  }"],
      "test_files": ["pkg/foo/foo_test.go"],
      "description": "Emptied the true-branch of an if statement on line 42.",
      "hint": "Add a test that enters the if-branch and asserts on its side-effect."
    }
  ]
}
```

| Field | Type | Description |
| :---- | :--- | :---------- |
| `id` | string | Stable hash of file + line + mutator — survives refactors that shift line numbers |
| `file` | string | Path to the mutated file, relative to the module root |
| `line` | int | Line number of the mutation |
| `mutator` | string | Mutator name (e.g. `branch/if`, `statement/return`) |
| `diff` | string | Unified diff of original vs mutated code |
| `context_lines` | []string | Surrounding source lines for orientation |
| `test_files` | []string | Test files in the same package |
| `description` | string | Human-readable description of what the mutator changed |
| `hint` | string | A concrete suggestion for a test that would kill this mutant |

### Usage with an LLM

```bash
go-mutesting --logger-agentic-json --quiet ./...
# Then feed go-mutesting-agentic.json to an LLM:
# "Here are the mutants my tests didn't catch. For each one,
#  write a Go test that would kill it."
```
