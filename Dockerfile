FROM golang:1.27-alpine AS builder
ARG VERSION=dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" -o /bin/olm-annotation-lint .

FROM alpine:3.24
LABEL org.opencontainers.image.source="https://github.com/openshift-kni/olm-annotation-lint"
LABEL org.opencontainers.image.description="Validates OLM annotations on Kubernetes resources"
LABEL org.opencontainers.image.licenses="Apache-2.0"
RUN adduser -D -h /home/linter linter
USER linter
COPY --from=builder /bin/olm-annotation-lint /bin/olm-annotation-lint
ENTRYPOINT ["/bin/olm-annotation-lint"]
