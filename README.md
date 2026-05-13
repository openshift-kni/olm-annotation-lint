# olm-annotation-lint

A GitHub Action and CLI tool that validates [OLM (Operator Lifecycle Manager)](https://olm.operatorframework.io/) annotations on Kubernetes resources.

OLM defines a small set of annotations across [`operator-framework/api`](https://github.com/operator-framework/api) and [`operator-framework/operator-lifecycle-manager`](https://github.com/operator-framework/operator-lifecycle-manager). This linter catches invalid, misspelled, or misused OLM annotations before they reach your cluster.

## What It Catches

| Rule | Severity | Example |
|---|---|---|
| Unknown OLM annotation | ERROR | `olm.operatorframework.io/bundle-install-timeout` (doesn't exist) |
| Wrong resource type | ERROR | `bundle-unpack-timeout` on a Subscription instead of OperatorGroup |
| Invalid value format | ERROR | `bundle-unpack-timeout: "not-a-duration"` |
| Wrong prefix | ERROR | `olm.operatorframework.io/` instead of `operatorframework.io/` |
| Case mismatch | ERROR | `OLM.providedAPIs` instead of `olm.providedAPIs` |
| Controller-managed annotation | WARNING | User setting `olm.operatorGroup` (OLM manages this) |

## Valid OLM Annotations

### User-Settable

| Annotation | Resource | Format |
|---|---|---|
| `operatorframework.io/bundle-unpack-timeout` | OperatorGroup | duration (e.g. `10m`) |
| `operatorframework.io/bundle-unpack-min-retry-interval` | OperatorGroup | duration |
| `operatorframework.io/priorityclass` | CatalogSource | string |
| `olm.catalogImageTemplate` | CatalogSource | template |
| `olm.skipRange` | ClusterServiceVersion | semver range |
| `olm.operatorframework.io/exclude-global-namespace-resolution` | Subscription | string |

### Controller-Managed (set by OLM, not users)

| Annotation | Resource |
|---|---|
| `olm.operatorGroup` | ClusterServiceVersion |
| `olm.operatorNamespace` | ClusterServiceVersion |
| `olm.targetNamespaces` | OperatorGroup |
| `olm.providedAPIs` | OperatorGroup |

## Usage as GitHub Action

```yaml
- uses: openshift-kni/olm-annotation-lint@v1
  with:
    path: "."
    exclude: "vendor,testdata"
    strict: "false"
```

### Inputs

| Input | Description | Default |
|---|---|---|
| `path` | Path or comma-separated paths to scan | `.` |
| `exclude` | Comma-separated paths to exclude | |
| `strict` | Treat warnings as errors | `false` |

## Usage as CLI

```bash
go install github.com/openshift-kni/olm-annotation-lint@latest

# Scan a directory
olm-annotation-lint --path /path/to/manifests

# Multiple paths, exclude vendor
olm-annotation-lint --path path1,path2 --exclude vendor

# JSON output
olm-annotation-lint --path . --format json

# Strict mode (warnings are errors)
olm-annotation-lint --path . --strict
```

### Output Formats

- `text` (default) — human-readable with file:line references
- `json` — structured output for tooling
- `github` — GitHub Actions annotation format
