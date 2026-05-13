FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/olm-annotation-lint .

FROM alpine:3.21
COPY --from=builder /bin/olm-annotation-lint /bin/olm-annotation-lint
ENTRYPOINT ["/bin/olm-annotation-lint"]
