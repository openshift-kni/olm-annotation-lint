BINARY := olm-annotation-lint
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

IMAGE ?= quay.io/bapalm/olm-annotation-lint
TAG ?= latest

.PHONY: build test lint clean scenario-test docker-build docker-lint

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./... -v

lint:
	golangci-lint run ./...

scenario-test: build
	@echo "=== Testing valid YAMLs ==="
	@./$(BINARY) --path testdata/valid && echo "PASS: all valid YAMLs accepted"
	@echo ""
	@echo "=== Testing invalid YAMLs (strict mode) ==="
	@for f in testdata/invalid/*.yaml; do \
		if ./$(BINARY) --path "$$f" --strict > /dev/null 2>&1; then \
			echo "FAIL: $$f should have been rejected"; \
			exit 1; \
		fi; \
		echo "PASS: $$f correctly rejected"; \
	done
	@echo ""
	@echo "=== Testing --version flag ==="
	@OUTPUT=$$(./$(BINARY) --version); \
	if [ -z "$$OUTPUT" ]; then \
		echo "FAIL: --version produced no output"; \
		exit 1; \
	fi; \
	echo "PASS: --version prints '$$OUTPUT'"
	@echo ""
	@echo "=== Testing --list-rules flag ==="
	@OUTPUT=$$(./$(BINARY) --list-rules); \
	if ! echo "$$OUTPUT" | grep -q "User-settable annotations"; then \
		echo "FAIL: --list-rules missing expected output"; \
		exit 1; \
	fi; \
	echo "PASS: --list-rules prints annotation list"

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(TAG) .

docker-lint: docker-build
	docker run --rm -v $(PWD):/workspace $(IMAGE):$(TAG) --path /workspace

clean:
	rm -f $(BINARY)
