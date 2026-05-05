# Makefile for the Gardener application.
#
# This file provides standardized commands for development, testing, and deployment.
# All targets are designed to be idempotent and safe to run multiple times.
#
# Usage:
#   make <target>
#
# Example:
#   make build    # Compile the application
#   make test     # Run unit tests with coverage
#   make lint     # Run linters for Go and frontend code
#
# Prerequisites:
#   - Go 1.22 or later
#   - Node.js 20 or later
#   - npm or compatible package manager
#   - golangci-lint (optional, for enhanced linting)

# Declare all targets as phony to avoid conflicts with files of the same name.
# This ensures targets always execute regardless of filesystem state.
.PHONY: help build run test lint clean install-deps frontend-build fmt

# Color codes for terminal output (ANSI escape sequences).
# Used to provide visual feedback during command execution.
BLUE := \033[34m
GREEN := \033[32m
YELLOW := \033[33m
RESET := \033[0m

# help: Display available commands with descriptions.
#
# This target prints a formatted list of all public make targets
# and their purposes. It serves as inline documentation for developers.
#
# Output: Formatted help text with color-coded target names.
help:
	@echo -e "$(BLUE)🌱 Огородник — доступные команды:$(RESET)"
	@echo -e "  $(GREEN)make install-deps$(RESET)  - Установить зависимости (Go + npm)"
	@echo -e "  $(GREEN)make frontend-build$(RESET) - Собрать фронтенд (Vite)"
	@echo -e "  $(GREEN)make build$(RESET)          - Полная сборка проекта"
	@echo -e "  $(GREEN)make run$(RESET)            - Запустить приложение"
	@echo -e "  $(GREEN)make test$(RESET)           - Запустить тесты (Go)"
	@echo -e "  $(GREEN)make lint$(RESET)           - Запустить линтеры (Go + JS)"
	@echo -e "  $(GREEN)make clean$(RESET)          - Очистить артефакты сборки"

# install-deps: Install all project dependencies.
#
# This target performs two steps:
#   1. Runs `go mod tidy` to sync Go module dependencies
#   2. Runs `npm install` in the frontend directory for JS dependencies
#
# Idempotent: Safe to run multiple times; only installs missing packages.
#
# Side effects:
#   - Updates go.sum with resolved module versions
#   - Creates/updates frontend/node_modules directory
install-deps:
	@echo -e "$(YELLOW)📦 Installing Go dependencies...$(RESET)"
	@go mod tidy
	@echo -e "$(YELLOW)📦 Installing frontend dependencies...$(RESET)"
	@cd frontend && npm install

# frontend-build: Compile the Svelte/Vite frontend application.
#
# This target runs the frontend build process, producing static assets
# in frontend/dist/ that are embedded into the Go binary via go:embed.
#
# Prerequisites:
#   - frontend/node_modules must exist (run `make install-deps` first)
#
# Output:
#   - frontend/dist/ directory containing optimized JS, CSS, and HTML
frontend-build:
	@echo -e "$(YELLOW)🎨 Building frontend...$(RESET)"
	@cd frontend && npm run build

# build: Compile the complete Gardener application.
#
# This is the primary build target. It performs:
#   1. frontend-build: Compile static assets
#   2. go build: Compile Go binary with embedded frontend
#
# Build flags:
#   -ldflags="-s -w": Strip debug symbols and DWARF table for smaller binary
#
# Output:
#   - ./gardener: Self-contained executable with embedded frontend
#
# Dependencies: frontend-build (declared as prerequisite)
build: frontend-build
	@echo -e "$(YELLOW)🔨 Building Go binary...$(RESET)"
	@go build -o gardener -ldflags="-s -w" .
	@echo -e "$(GREEN)✅ Build complete: ./gardener$(RESET)"

# run: Build and execute the Gardener application.
#
# This target combines build and execution for local development.
# It first ensures the binary is up-to-date, then launches it.
#
# Behavior:
#   - Builds the application if not already built
#   - Starts HTTP server on 127.0.0.1:8080
#   - Attempts to open default browser to application URL
#
# Note: The server runs in the foreground; use Ctrl+C to stop.
#
# Dependencies: build (declared as prerequisite)
run: build
	@echo -e "$(GREEN)🚀 Starting Gardener...$(RESET)"
	@./gardener

# test: Execute Go unit tests with race detection and coverage.
#
# This target runs all tests in the internal/ package tree with:
#   -v: Verbose output showing individual test names
#   -race: Enable data race detector for concurrency safety
#   -cover: Generate code coverage statistics
#
# Scope: ./internal/... (API handlers and database layer)
#
# Output:
#   - Test results with pass/fail status per test function
#   - Coverage percentage per package
#
# Exit code:
#   - 0: All tests passed
#   - 1: One or more tests failed
test:
	@echo -e "$(YELLOW)🧪 Running Go tests...$(RESET)"
	@go test -v -race -cover ./internal/...
	@echo -e "$(GREEN)✅ Tests passed$(RESET)"

# lint: Run code quality checks for Go and frontend sources.
#
# This target performs two linting stages:
#
# 1. Go linting:
#    - go fmt: Auto-format Go source files (modifies files in place)
#    - golangci-lint: Run comprehensive static analysis (if installed)
#
# 2. Frontend linting:
#    - eslint: Check JavaScript/Svelte files for errors and style issues
#
# Non-fatal: Missing golangci-lint produces a warning but does not fail.
#
# Side effects:
#   - go fmt may modify source files to match standard formatting
#
# Exit code:
#   - 0: No lint errors found
#   - 1: Lint errors detected (fails CI/CD pipelines)
lint:
	@echo -e "$(YELLOW)🔍 Running Go linter...$(RESET)"
	@go fmt ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo -e "$(YELLOW)⚠️ golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(RESET)"; \
	fi
	@echo -e "$(YELLOW)🔍 Running frontend linter...$(RESET)"
	@cd frontend && npx eslint src/
	@echo -e "$(GREEN)✅ Lint passed$(RESET)"

# clean: Remove build artifacts and cached data.
#
# This target resets the project to a clean state by removing:
#   - gardener: Compiled Go binary
#   - frontend/dist/: Built frontend assets
#   - Go build cache and module cache
#
# Safe to run: Does not delete source code, configuration, or user data.
# User data (~/.gardener/) is never affected by this target.
#
# Use case:
#   - Before a fresh build to ensure no stale artifacts
#   - To free disk space occupied by build caches
clean:
	@echo -e "$(YELLOW)🧹 Cleaning...$(RESET)"
	@rm -f gardener
	@rm -rf frontend/dist
	@go clean -cache -modcache
	@echo -e "$(GREEN)✅ Clean complete$(RESET)"

# fmt: Format all source files according to language standards.
#
# This target applies automatic formatting to:
#   - Go files: via `go fmt` (enforces standard Go style)
#   - Frontend files: via `prettier` (enforces consistent JS/Svelte style)
#
# Idempotent: Running multiple times produces the same result.
#
# Side effects:
#   - Modifies source files in place to match formatting rules
#   - Does not change code semantics, only whitespace and style
#
# Note: Frontend formatting errors are suppressed (|| true) to avoid
# failing the target if prettier is not installed or encounters issues.
fmt:
	@echo -e "$(YELLOW)🎨 Formatting code...$(RESET)"
	@go fmt ./...
	@cd frontend && npx prettier --write src/ 2>/dev/null || true
	@echo -e "$(GREEN)✅ Formatting complete$(RESET)"