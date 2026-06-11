# Installation

## Binary Download

No Go toolchain or clone required. Download the latest release binary:

```bash
curl -sL https://github.com/openshift-kni/olm-annotation-lint/releases/latest/download/olm-annotation-lint-linux-amd64 -o olm-annotation-lint
chmod +x olm-annotation-lint
./olm-annotation-lint --path .
```

Available binaries: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`.

## Container (Docker/Podman)

```bash
docker run --rm -v $(pwd):/workspace quay.io/bapalm/olm-annotation-lint:latest --path /workspace
```

## Go Install

```bash
go install github.com/openshift-kni/olm-annotation-lint@latest
```

## Makefile Integration

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
make run-annotation-lint
make run-annotation-lint OAL_LINT_PATH=manifests OAL_LINT_ARGS="--strict --exclude vendor"
make run-annotation-lint OAL_VERSION=v1.2.0
```

Add `.local/` to your `.gitignore` to keep the cached binary out of version control.

---

Next: [CLI Usage](cli-usage.md)
