# olm-annotation-lint

A GitHub Action and CLI tool that validates [OLM (Operator Lifecycle Manager)](https://olm.operatorframework.io/) annotations on Kubernetes resources. Catches invalid, misspelled, or misused OLM annotations before they reach your cluster.

## Key Features

- **Unknown Annotation Detection** — Flags OLM-prefixed annotations that don't exist
- **Resource Type Validation** — Ensures annotations are on the correct resource kind
- **Value Format Checking** — Validates durations, semver ranges, JSON, and templates
- **Case Mismatch Detection** — Catches `OLM.providedAPIs` vs `olm.providedAPIs`
- **Controller-Managed Warnings** — Warns when users set annotations OLM manages
- **Multiple Output Formats** — Text, JSON, GitHub Actions, JUnit, and SARIF
- **Stdin Support** — Pipe `kubectl get` output directly for validation

## Quick Start

```bash
curl -sL https://github.com/openshift-kni/olm-annotation-lint/releases/latest/download/olm-annotation-lint-linux-amd64 -o olm-annotation-lint
chmod +x olm-annotation-lint
./olm-annotation-lint --path .
```

## GitHub Action

```yaml
- uses: openshift-kni/olm-annotation-lint@v1
  with:
    path: "."
    exclude: "vendor,testdata"
    format: "github"
```

## Guides

| Guide | Description |
|-------|-------------|
| [Installation](docs/installation.md) | Binary download, container, and Makefile integration |
| [CLI Usage](docs/cli-usage.md) | Flags, configuration file, exit codes, output formats |
| [GitHub Action](docs/github-action.md) | Action inputs and workflow examples |
| [Annotations Reference](docs/annotations.md) | All valid OLM annotations and validation rules |
| [Adopters](ADOPTERS.md) | Repos that could run this linter in CI |

## Prerequisites

- Go 1.26+ (building from source only)

## Development

```bash
make build          # Build binary
make test           # Run unit tests
make coverage       # Run tests with coverage (mirrors CI)
make lint           # Run linter
make docker-build   # Build container image
make docker-lint    # Lint via container
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
