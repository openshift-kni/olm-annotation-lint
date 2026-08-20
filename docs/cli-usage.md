# CLI Usage

## Examples

```bash
olm-annotation-lint --path /path/to/manifests
olm-annotation-lint -p path1,path2 -e vendor
olm-annotation-lint -p . -f json
olm-annotation-lint -p . -s
olm-annotation-lint -p . -a olm.operatorframework.io/bundle-install-timeout
olm-annotation-lint -v
olm-annotation-lint -l
kubectl get csv -n openshift-operators -o yaml | olm-annotation-lint -p -
```

## Flags

All flags support both long and short forms:

| Long | Short | Description |
|------|-------|-------------|
| `--path` | `-p` | Path or comma-separated paths to scan |
| `--exclude` | `-e` | Comma-separated paths to exclude |
| `--allow` | `-a` | Comma-separated annotation keys to allow |
| `--strict` | `-s` | Treat warnings as errors |
| `--format` | `-f` | Output format: text, json, github, junit |
| `--config` | `-c` | Path to config file |
| `--version` | `-v` | Print version and exit |
| `--list-rules` | `-l` | List all known OLM annotations and exit |

## Configuration File

Create a `.olm-lint.yaml` in your project root to store default settings:

```yaml
path:
  - "manifests"
  - "deploy"
exclude:
  - "vendor"
  - "testdata"
allow:
  - "olm.operatorframework.io/bundle-install-timeout"
strict: false
```

Fields accept either a list or a comma-separated string (e.g., `exclude: "vendor,testdata"`).

The config file is auto-discovered in the current directory. Use `--config` to specify a
custom path. CLI flags always take precedence over config file values.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | No errors (warnings may be present) |
| `1` | Errors found (or warnings in `--strict` mode) |
| `2` | Runtime error (invalid path, unreadable file, bad flags) |

## Output Formats

- `text` (default) — human-readable with file:line references
- `json` — structured report with summary counts and violation details
- `github` — GitHub Actions annotation format
- `junit` — JUnit XML for CI dashboards (errors are failures; warnings are `system-err`)
- `sarif` — SARIF 2.1.0 for the GitHub Security tab

### JSON Format

```json
{
  "version": "v1.0.10",
  "summary": {
    "errors": 1,
    "warnings": 1,
    "total": 2
  },
  "violations": [
    {
      "file": "manifests/csv.yaml",
      "line": 12,
      "annotation": "olm.fakeAnnotation",
      "kind": "ClusterServiceVersion",
      "severity": "error",
      "rule": "unknown-annotation",
      "message": "unknown OLM annotation"
    }
  ]
}
```

---

Next: [GitHub Action](github-action.md)
