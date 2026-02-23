# ============================================================================
# Draupnir — Sindri Instance Agent
# ============================================================================

.PHONY: build build-all test fmt fmt-check vet lint audit install clean ci

# ── Color codes ──────────────────────────────────────────────────────────────
BLUE    := \033[0;34m
GREEN   := \033[0;32m
YELLOW  := \033[1;33m
RED     := \033[0;31m
BOLD    := \033[1m
RESET   := \033[0m

GO      := go
BINARY  := sindri-agent

# ============================================================================
# Build
# ============================================================================

build:
	@echo "$(BLUE)Building agent for current platform (static)...$(RESET)"
	@mkdir -p dist
	CGO_ENABLED=0 $(GO) build -ldflags "-s -w" -o dist/$(BINARY) ./cmd/agent
	@echo "$(GREEN)✓ Agent built: dist/$(BINARY)$(RESET)"

build-all:
	@echo "$(BLUE)Cross-compiling agent for all platforms...$(RESET)"
	@mkdir -p dist
	GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 $(GO) build -ldflags "-s -w" -o dist/$(BINARY)-linux-amd64 ./cmd/agent
	GOOS=linux  GOARCH=arm64 CGO_ENABLED=0 $(GO) build -ldflags "-s -w" -o dist/$(BINARY)-linux-arm64 ./cmd/agent
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -ldflags "-s -w" -o dist/$(BINARY)-darwin-amd64 ./cmd/agent
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build -ldflags "-s -w" -o dist/$(BINARY)-darwin-arm64 ./cmd/agent
	@echo "$(GREEN)✓ Cross-compiled binaries:$(RESET)"
	@ls -lh dist/

# ============================================================================
# Test & Quality
# ============================================================================

test:
	@echo "$(BLUE)Running unit tests...$(RESET)"
	$(GO) test ./... -count=1 -timeout 120s -race
	@echo "$(GREEN)✓ Tests passed$(RESET)"

fmt:
	@echo "$(BLUE)Formatting Go code...$(RESET)"
	gofmt -w .
	@echo "$(GREEN)✓ Code formatted$(RESET)"

fmt-check:
	@echo "$(BLUE)Checking Go formatting...$(RESET)"
	@UNFORMATTED=$$(gofmt -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "$(RED)✗ Unformatted files:$(RESET)"; \
		echo "$$UNFORMATTED"; \
		echo "  Run: make fmt"; \
		exit 1; \
	fi
	@echo "$(GREEN)✓ Formatting check passed$(RESET)"

vet:
	@echo "$(BLUE)Running go vet...$(RESET)"
	$(GO) vet ./...
	@echo "$(GREEN)✓ go vet passed$(RESET)"

lint:
	@echo "$(BLUE)Running golangci-lint...$(RESET)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)golangci-lint not installed. Falling back to go vet.$(RESET)"; \
		$(GO) vet ./...; \
	else \
		golangci-lint run ./...; \
	fi
	@echo "$(GREEN)✓ Lint passed$(RESET)"

audit:
	@echo "$(BLUE)Running vulnerability scan (govulncheck)...$(RESET)"
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "$(YELLOW)govulncheck not installed. Skipping.$(RESET)"; \
	else \
		govulncheck ./...; \
		echo "$(GREEN)✓ Vulnerability scan passed$(RESET)"; \
	fi

# ============================================================================
# Install & Clean
# ============================================================================

install: build
	@echo "$(BLUE)Installing $(BINARY) to ~/.local/bin...$(RESET)"
	@mkdir -p ~/.local/bin
	@cp dist/$(BINARY) ~/.local/bin/$(BINARY)
	@chmod +x ~/.local/bin/$(BINARY)
	@echo "$(GREEN)✓ Installed: ~/.local/bin/$(BINARY)$(RESET)"

clean:
	@echo "$(BLUE)Cleaning build artifacts...$(RESET)"
	@rm -rf dist
	@echo "$(GREEN)✓ Artifacts cleaned$(RESET)"

ci: vet fmt-check test build-all
	@echo "$(GREEN)$(BOLD)✓ CI pipeline passed$(RESET)"
