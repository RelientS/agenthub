# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go module files first for layer caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /app/bin/agenthub-server \
    ./cmd/server/main.go

# Build the CLI binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/agenthub-cli \
    ./cli/...

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S agenthub && adduser -S agenthub -G agenthub

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/agenthub-server /app/agenthub-server
COPY --from=builder /app/bin/agenthub-cli /app/agenthub-cli

# Copy migrations for runtime migration support
COPY --from=builder /app/migrations /app/migrations

# Set ownership
RUN chown -R agenthub:agenthub /app

USER agenthub

EXPOSE 8080

ENTRYPOINT ["/app/agenthub-server"]
