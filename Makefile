BINARY := olm-annotation-lint
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

IMAGE ?= quay.io/bapalm/olm-annotation-lint
TAG ?= latest

.PHONY: build test lint coverage clean scenario-test docker-build docker-lint

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./... -v

coverage:
	go test -race ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | grep ^total

lint:
	golangci-lint run ./...

scenario-test: build
	BINARY=./$(BINARY) bash hack/scenario-test.sh

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(TAG) .

docker-lint: docker-build
	docker run --rm -v $(PWD):/workspace $(IMAGE):$(TAG) --path /workspace

clean:
	rm -f $(BINARY)
