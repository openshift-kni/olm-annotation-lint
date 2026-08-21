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
| `--exclude` | `-e` | Comma-separated directory names or file globs to skip |
| `--allow` | `-a` | Comma-separated annotation keys to allow. A trailing `*` matches a prefix (for example `olm.*`). |
| `--strict` | `-s` | Treat warnings as errors |
| `--format` | `-f` | Output format: text, json, github, junit |
| `--config` | `-c` | Path to config file |
| `--output` | `-o` | Write results to a file |
| `--timeout` | | Maximum duration (e.g. `30s`, `5m`); 0 means no timeout |
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
format: text
rules:
  olm.operatorGroup:
    enabled: false
  unknown-annotation:
    severity: warning
```

Fields accept either a list or a comma-separated string (e.g., `exclude: "vendor,testdata"`).

The `rules` map is keyed by annotation name (`olm.skipRange`) or rule ID
(`unknown-annotation`, `case-mismatch`, `controller-managed`, and so on).
Set `enabled: false` to suppress matching violations, or `severity` to
`error`, `warning`, or `info`.

The config file is auto-discovered in the current directory. Use `--config` to specify a
custom path. CLI flags always take precedence over config file values.

## Inline ignore directives

Suppress violations for a specific annotation with a YAML comment:

```yaml
metadata:
  annotations:
    # olm-annotation-lint: ignore
    olm.custom.annotation: "value"
    olm.operatorGroup: og-test  # olm-annotation-lint: ignore controller-managed
```

`# olm-annotation-lint: ignore` skips all rules for that key. Add one or more
rule IDs (from JSON/JUnit/SARIF output) to ignore only those rules.

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
      "name": "test-operator.v1.0.0",
      "severity": "error",
      "rule": "unknown-annotation",
      "message": "unknown OLM annotation"
    }
  ]
}
```

---

Next: [GitHub Action](github-action.md)
