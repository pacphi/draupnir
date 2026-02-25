# ============================================================================
# Draupnir — Sindri Instance Agent
# ============================================================================

.PHONY: help build build-all test fmt fmt-check vet lint audit deadcode \
	deps-upgrade deps-outdated \
	install clean ci hooks

.DEFAULT_GOAL := help

# ── Color codes ──────────────────────────────────────────────────────────────
BLUE    := \033[0;34m
GREEN   := \033[0;32m
YELLOW  := \033[1;33m
RED     := \033[0;31m
BOLD    := \033[1m
RESET   := \033[0m

# ============================================================================
# Help
# ============================================================================

help:
	@echo "$(BOLD)$(BLUE)╔════════════════════════════════════════════════════════════════════╗$(RESET)"
	@echo "$(BOLD)$(BLUE)║                        Draupnir Makefile                           ║$(RESET)"
	@echo "$(BOLD)$(BLUE)╚════════════════════════════════════════════════════════════════════╝$(RESET)"
	@echo ""
	@echo "$(BOLD)Quick Start:$(RESET)"
	@echo "  make build             - Build for current platform"
	@echo "  make test              - Run unit tests"
	@echo "  make ci                - Run full CI pipeline"
	@echo "  make install           - Install to ~/.local/bin"
	@echo ""
	@echo "$(BOLD)$(BLUE)═══ Build ═══════════════════════════════════════════════════════════$(RESET)"
	@echo "  build                  - Build agent for current platform (static)"
	@echo "  build-all              - Cross-compile for linux/darwin (amd64 + arm64)"
	@echo "  install                - Build and install to ~/.local/bin"
	@echo "  clean                  - Remove build artifacts (dist/)"
	@echo ""
	@echo "$(BOLD)$(BLUE)═══ Test & Quality ══════════════════════════════════════════════════$(RESET)"
	@echo "  test                   - Run unit tests with race detection"
	@echo "  fmt                    - Format Go code (gofmt -w)"
	@echo "  fmt-check              - Check formatting without changes"
	@echo "  vet                    - Run go vet"
	@echo "  lint                   - Run golangci-lint (falls back to go vet)"
	@echo "  audit                  - Run govulncheck vulnerability scan"
	@echo "  deadcode               - Scan for dead code"
	@echo ""
	@echo "$(BOLD)$(BLUE)═══ Dependency Management ═══════════════════════════════════════════$(RESET)"
	@echo "  deps-upgrade           - Upgrade Go dependencies to latest"
	@echo "  deps-outdated          - Check for outdated Go modules"
	@echo ""
	@echo "$(BOLD)$(BLUE)═══ CI & Tooling ════════════════════════════════════════════════════$(RESET)"
	@echo "  ci                     - Run full CI pipeline (vet + fmt-check + test + build-all)"
	@echo "  hooks                  - Install git hooks (.githooks/pre-commit)"
	@echo ""

GO      := go
BINARY  := draupnir
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# ============================================================================
# Build
# ============================================================================

build:
	@echo "$(BLUE)Building agent for current platform (static)...$(RESET)"
	@mkdir -p dist
	CGO_ENABLED=0 $(GO) build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/$(BINARY) ./cmd/agent
	@echo "$(GREEN)✓ Agent built: dist/$(BINARY)$(RESET)"

build-all:
	@echo "$(BLUE)Cross-compiling agent for all platforms...$(RESET)"
	@mkdir -p dist
	GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 $(GO) build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/$(BINARY)-linux-amd64 ./cmd/agent
	GOOS=linux  GOARCH=arm64 CGO_ENABLED=0 $(GO) build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/$(BINARY)-linux-arm64 ./cmd/agent
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GO) build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/$(BINARY)-darwin-amd64 ./cmd/agent
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GO) build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/$(BINARY)-darwin-arm64 ./cmd/agent
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
# Dependency Management
# ============================================================================

deps-upgrade:
	@echo "$(BLUE)Upgrading Go dependencies to latest...$(RESET)"
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "$(GREEN)✓ Go dependencies upgraded$(RESET)"

deps-outdated:
	@echo "$(BLUE)Checking for outdated Go modules...$(RESET)"
	$(GO) list -m -u all
	@echo "$(GREEN)✓ Outdated check complete$(RESET)"

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

# ============================================================================
# Dead Code Detection
# ============================================================================

deadcode:
	@echo "$(BLUE)Scanning for dead code (golang.org/x/tools/cmd/deadcode)...$(RESET)"
	@if ! command -v deadcode >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing deadcode...$(RESET)"; \
		go install golang.org/x/tools/cmd/deadcode@latest; \
	fi
	deadcode ./... || true
	@echo "$(GREEN)✓ Dead code scan complete$(RESET)"

# ============================================================================
# Git Hooks
# ============================================================================

hooks:
	@echo "$(BLUE)Installing git hooks...$(RESET)"
	git config core.hooksPath .githooks
	@echo "$(GREEN)✓ Git hooks installed (.githooks/pre-commit)$(RESET)"
