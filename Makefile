.PHONY: build test lint vet fmt clean release docker deploy

# ——————————————————————————————————————————————————————
# StackWatch Makefile — single source of truth for build & test
# ——————————————————————————————————————————————————————

BUILD_DIR := bin
CMDS := api-gateway agent web-terminal truenas-connector

.DEFAULT_GOAL := all

all: build test

# Build all binaries
build:
	@echo "Building binaries..."
	@mkdir -p $(BUILD_DIR)
	@for cmd in $(CMDS); do \
		echo "  Building cmd/$$cmd"; \
		go build -o $(BUILD_DIR)/$$cmd ./cmd/$$cmd; \
	done
	@echo "Done. Binaries in $(BUILD_DIR)/"

# Build for cross-compilation (linux/arm64 + linux/amd64 + windows/amd64 + darwin/amd64)
release: clean
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)/release
	@for cmd in $(CMDS); do \
		echo "  $$cmd: linux/amd64"; \
		GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/release/$$cmd-linux-amd64 ./cmd/$$cmd; \
		echo "  $$cmd: linux/arm64"; \
		GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/release/$$cmd-linux-arm64 ./cmd/$$cmd; \
		echo "  $$cmd: windows/amd64"; \
		GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/release/$$cmd-windows-amd64.exe ./cmd/$$cmd; \
		echo "  $$cmd: darwin/amd64"; \
		GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_DIR)/release/$$cmd-darwin-amd64 ./cmd/$$cmd; \
	done
	@cd $(BUILD_DIR)/release && sha256sum * > SHA256SUMS.txt
	@echo "Release ready in $(BUILD_DIR)/release/"

# Run all tests (fast, then full with race)
test:
	go test ./... -count=1 -timeout=60s

test-race:
	go test ./... -race -count=1 -timeout=120s

test-cover:
	go test ./... -count=1 -coverprofile=coverage.out -timeout=60s
	go tool cover -func=coverage.out | tail -1

# Linting
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format check
fmt:
	@echo "Checking gofmt..."
	@! gofmt -l ./internal ./cmd | grep -q . || (echo "Run: gofmt -w ./internal ./cmd" && exit 1)
	@echo "All files are gofmt clean"

vet:
	go vet ./...

# Static analysis combo
check: fmt vet lint test

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)/
	go clean -testcache

# Docker images
docker-build:
	docker compose -f deploy/docker-compose.yml build api-gateway agent web-terminal

# Deploy to .115 (requires SSH config set up)
deploy: build
	@echo "Deploying to .115..."
	@scp $(BUILD_DIR)/api-gateway root@192.168.0.115:/opt/stackwatch/bin/api-gateway-linux
	@ssh root@192.168.0.115 "systemctl restart stackwatch-api-gateway && sleep 2 && curl -sf http://127.0.0.1:8080/health"
	@echo "Deployed"

# Run local dev instance (with .env from web/dotenv or inline)
dev: build
	@./$(BUILD_DIR)/api-gateway
