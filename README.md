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

## Quick Start

No Go toolchain or clone required. Pick whichever method suits your environment:

**Binary download:**

```bash
# Download the latest release binary (adjust OS/arch as needed)
curl -sL https://github.com/openshift-kni/olm-annotation-lint/releases/latest/download/olm-annotation-lint-linux-amd64 -o olm-annotation-lint
chmod +x olm-annotation-lint
./olm-annotation-lint --path .
```

Available binaries: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`.

**Container (Docker/Podman):**

```bash
docker run --rm -v $(pwd):/workspace quay.io/bapalm/olm-annotation-lint:latest --path /workspace
```

### Makefile Integration

Add a `run-annotation-lint` target to any repo. It auto-detects OS/arch, downloads the
binary on first run (cached in `.local/bin/`), and falls back to the container image if
the download fails:

```makefile
OAL_VERSION ?= latest
OAL_LINT_PATH ?= .
OAL_LINT_ARGS ?=
OAL_BIN_DIR := .local/bin
OAL_BIN := $(OAL_BIN_DIR)/olm-annotation-lint
OAL_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
OAL_ARCH := $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')
OAL_IMAGE := quay.io/bapalm/olm-annotation-lint:latest

ifeq ($(OAL_VERSION),latest)
OAL_URL := https://github.com/openshift-kni/olm-annotation-lint/releases/latest/download/olm-annotation-lint-$(OAL_OS)-$(OAL_ARCH)
else
OAL_URL := https://github.com/openshift-kni/olm-annotation-lint/releases/download/$(OAL_VERSION)/olm-annotation-lint-$(OAL_OS)-$(OAL_ARCH)
endif

$(OAL_BIN):
	@mkdir -p $(OAL_BIN_DIR)
	@echo "Downloading olm-annotation-lint ($(OAL_OS)/$(OAL_ARCH))..."
	@curl -sfL $(OAL_URL) -o $(OAL_BIN) && chmod +x $(OAL_BIN) \
		|| (echo "Download failed, falling back to container image"; rm -f $(OAL_BIN))

.PHONY: run-annotation-lint
run-annotation-lint: $(OAL_BIN)
	@if [ -x $(OAL_BIN) ]; then \
		$(OAL_BIN) --path $(OAL_LINT_PATH) $(OAL_LINT_ARGS); \
	else \
		docker run --rm -v $(PWD):/workspace $(OAL_IMAGE) --path /workspace/$(OAL_LINT_PATH) $(OAL_LINT_ARGS); \
	fi
```

Then run:

```bash
# Lint current directory
make run-annotation-lint

# Lint a specific path with extra flags
make run-annotation-lint OAL_LINT_PATH=manifests OAL_LINT_ARGS="--strict --exclude vendor"

# Pin to a specific version
make run-annotation-lint OAL_VERSION=v1.2.0

# Force re-download
rm -rf .local/bin && make run-annotation-lint
```

Add `.local/` to your `.gitignore` to keep the cached binary out of version control.

## Usage as GitHub Action

```yaml
- uses: openshift-kni/olm-annotation-lint@v1
  with:
    path: "."
    exclude: "vendor,testdata"
    strict: "false"
    allow: "olm.operatorframework.io/bundle-install-timeout"
```

### Inputs

| Input | Description | Default |
|---|---|---|
| `path` | Path or comma-separated paths to scan | `.` |
| `exclude` | Comma-separated paths to exclude | |
| `strict` | Treat warnings as errors | `false` |
| `allow` | Comma-separated annotation keys to bypass unknown annotation errors | |

## Usage as CLI

```bash
go install github.com/openshift-kni/olm-annotation-lint@latest

# Scan a directory
olm-annotation-lint --path /path/to/manifests
olm-annotation-lint -p /path/to/manifests

# Multiple paths, exclude vendor
olm-annotation-lint -p path1,path2 -e vendor

# JSON output
olm-annotation-lint -p . -f json

# Strict mode (warnings are errors)
olm-annotation-lint -p . -s

# Allow specific annotations not yet in the hardcoded list
olm-annotation-lint -p . -a olm.operatorframework.io/bundle-install-timeout

# Print version
olm-annotation-lint -v

# List all known OLM annotations
olm-annotation-lint -l

# Read from stdin (use -p -)
kubectl get csv -n openshift-operators -o yaml | olm-annotation-lint -p -
cat manifest.yaml | olm-annotation-lint -p -
```

All flags support both long and short forms:

| Long | Short | Description |
|------|-------|-------------|
| `--path` | `-p` | Path or comma-separated paths to scan |
| `--exclude` | `-e` | Comma-separated paths to exclude |
| `--allow` | `-a` | Comma-separated annotation keys to allow |
| `--strict` | `-s` | Treat warnings as errors |
| `--format` | `-f` | Output format: text, json, github |
| `--config` | `-c` | Path to config file |
| `--version` | `-v` | Print version and exit |
| `--list-rules` | `-l` | List all known OLM annotations and exit |

### Configuration File

Create a `.olm-lint.yaml` in your project root to store default settings:

```yaml
# .olm-lint.yaml
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

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | No errors (warnings may be present) |
| `1` | Errors found (or warnings in `--strict` mode) |
| `2` | Runtime error (invalid path, unreadable file, bad flags) |

### Output Formats

- `text` (default) — human-readable with file:line references
- `json` — structured output for tooling
- `github` — GitHub Actions annotation format

## Building from Source

```bash
# Build binary
make build
./olm-annotation-lint --path .

# Build and run via container
make docker-build
make docker-lint
```
