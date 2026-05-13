BINARY := olm-annotation-lint

.PHONY: build test lint clean scenario-test

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

clean:
	rm -f $(BINARY)
