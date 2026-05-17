# Configuration File

go-mutesting can be configured via a YAML file. Pass it with `--config <path>`. No default config file is loaded automatically.

## Schema

```yaml
skip_without_test: true
skip_with_build_tags: true
json_output: false
html_output: false
silent_mode: false
min_msi: 0
min_covered_msi: 0
exclude_dirs: []
```

## Fields

| Field | Type | Default | Description |
| :---- | :--- | :------ | :---------- |
| `skip_without_test` | bool | `true` | Skip source files that have no corresponding `_test.go` file |
| `skip_with_build_tags` | bool | `true` | Skip test files that have build tags |
| `json_output` | bool | `false` | Write a full mutation report to `report.json` |
| `html_output` | bool | `false` | Write an HTML report to `go-mutesting-report.html` |
| `silent_mode` | bool | `false` | Suppress per-mutation output (summary only) |
| `min_msi` | float | `0` | Minimum overall MSI (0–100); `0` disables the gate |
| `min_covered_msi` | float | `0` | Minimum covered-code MSI (0–100); `0` disables the gate |
| `exclude_dirs` | []string | `[]` | Directory path prefixes to exclude from mutation |

## Notes

- CLI flags override config file values.
- Unknown keys in the config file cause an error (strict parsing).
- `exclude_dirs` values are matched as path *prefixes*, not globs. For example, `vendor` excludes any path starting with `vendor`.

## Example

```yaml
skip_without_test: true
min_msi: 70
min_covered_msi: 80
exclude_dirs:
  - vendor
  - testdata
```
