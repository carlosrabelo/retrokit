MAKEFLAGS += --no-print-directory

.DEFAULT_GOAL := help

.PHONY: build clean fmt help install lint quality test uninstall

BINARY_NAME := retrokit

build: ## Build binary
	@./.make/build.sh

clean: ## Remove build artifacts
	@./.make/clean.sh

fmt: ## Format Go source code
	@go fmt ./...

help: ## Show available targets
	@echo "$(BINARY_NAME) - Available targets"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*## "} {printf "  %-15s %s\n", $$1, $$2}'

install: build ## Install binary to ~/.local/bin
	@./.make/install.sh

lint: ## Run Go vet
	@go vet ./...

quality: ## Run all quality checks
	@./.make/quality.sh

test: ## Run tests
	@./.make/test.sh

uninstall: ## Remove binary from ~/.local/bin
	@./.make/uninstall.sh
