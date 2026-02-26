.PHONY: all build run test migrate docker-up docker-down lint clean cli-build mcp-build web-dev help

# Variables
BINARY_NAME=agenthub-server
CLI_BINARY=agenthub-cli
BUILD_DIR=./bin
CMD_DIR=./cmd/server
CLI_DIR=./cli
MCP_DIR=./mcp-server
WEB_DIR=./web
MIGRATIONS_DIR=./migrations
DATABASE_URL?=postgres://agenthub:agenthub_secret@localhost:5432/agenthub?sslmode=disable
GO=go
GOFLAGS=-v
LDFLAGS=-w -s

# Default target
all: build cli-build

## build: Build the server binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## run: Run the server locally
run:
	$(GO) run $(CMD_DIR)/main.go

## test: Run all tests
test:
	$(GO) test ./... -v -race -cover

## migrate: Run database migrations
migrate:
	@echo "Running migrations against $(DATABASE_URL)..."
	@for f in $(MIGRATIONS_DIR)/*.sql; do \
		echo "Applying $$f..."; \
		psql "$(DATABASE_URL)" -f "$$f"; \
	done
	@echo "Migrations complete."

## docker-up: Start all services with Docker Compose
docker-up:
	docker-compose up -d
	@echo "Services starting... Use 'docker-compose logs -f' to follow logs."

## docker-down: Stop all Docker Compose services
docker-down:
	docker-compose down

## lint: Run linter
lint:
	golangci-lint run ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	$(GO) clean
	@echo "Clean complete."

## cli-build: Build the CLI tool
cli-build:
	@echo "Building $(CLI_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(CLI_BINARY) $(CLI_DIR)/...
	@echo "CLI build complete: $(BUILD_DIR)/$(CLI_BINARY)"

## mcp-build: Build the MCP server
mcp-build:
	@echo "Building MCP server..."
	cd $(MCP_DIR) && npm install && npm run build
	@echo "MCP server build complete: $(MCP_DIR)/dist/"

## web-dev: Start the web development server
web-dev:
	cd $(WEB_DIR) && npm install && npm run dev

## help: Show this help message
help:
	@echo "AgentHub - Multi-Agent Collaboration Platform"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
