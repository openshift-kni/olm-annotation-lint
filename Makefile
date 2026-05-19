BINARY := olm-annotation-lint

IMAGE ?= quay.io/bapalm/olm-annotation-lint
TAG ?= latest

.PHONY: build test lint clean scenario-test docker-build docker-lint

build:
	go build -o $(BINARY) .

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

docker-build:
	docker build -t $(IMAGE):$(TAG) .

docker-lint: docker-build
	docker run --rm -v $(PWD):/workspace $(IMAGE):$(TAG) --path /workspace

clean:
	rm -f $(BINARY)
