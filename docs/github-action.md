# GitHub Action

## Usage

```yaml
- uses: openshift-kni/olm-annotation-lint@v1
  with:
    path: "."
    exclude: "vendor,testdata"
    strict: "false"
    allow: "olm.operatorframework.io/bundle-install-timeout"
    format: "github"
```

## Inputs

| Input | Description | Default |
|---|---|---|
| `path` | Path or comma-separated paths to scan | `.` |
| `exclude` | Comma-separated paths to exclude | |
| `strict` | Treat warnings as errors | `false` |
| `allow` | Comma-separated annotation keys to bypass unknown annotation errors | |
| `format` | Output format: `text`, `json`, `github` | `github` |

## Full Workflow Example

```yaml
name: OLM Lint
on:
  pull_request:
    paths:
      - '**.yaml'
      - '**.yml'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: openshift-kni/olm-annotation-lint@v1
        with:
          path: "."
          exclude: "vendor,testdata"
          format: "github"
```

The `github` format produces inline annotations on the PR diff for each violation.

---

Next: [Annotations Reference](annotations.md)
