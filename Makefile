# Get the latest git tag
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
# Get the next version by incrementing the patch number
NEXT_VERSION := $(shell echo $(VERSION) | awk -F. '{$$NF = $$NF + 1;} 1' | sed 's/ /./g')

.PHONY: help test lint coverage clean build examples security

# Help target that lists all available targets with descriptions
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the project
	go build -v ./...

test: ## Run tests
	go test -v -race ./...

coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt

lint: ## Run linters
	golangci-lint run 2>&1

security: ## Run security scan
	gosec -quiet ./... 2>&1

# Combined quality checks
check: lint security test ## Run all quality checks (lint, security, test)

clean: ## Clean build artifacts
	go clean
	rm -f coverage.txt
	rm -f userprefs

examples: ## Build example applications
	go build -v ./examples/...

# Release targets
release-major: ## Create a new major release (v1.0.0 -> v2.0.0)
	$(eval NEW_VERSION := $(shell echo $(VERSION) | awk -F. '{$$1 = $$1 + 1; $$2 = 0; $$3 = 0;} 1' | sed 's/ /./g'))
	@echo "Creating major release $(NEW_VERSION)"
	git tag -a $(NEW_VERSION) -m "Release $(NEW_VERSION)"
	git push origin $(NEW_VERSION)

release-minor: ## Create a new minor release (v1.0.0 -> v1.1.0)
	$(eval NEW_VERSION := $(shell echo $(VERSION) | awk -F. '{$$2 = $$2 + 1; $$3 = 0;} 1' | sed 's/ /./g'))
	@echo "Creating minor release $(NEW_VERSION)"
	git tag -a $(NEW_VERSION) -m "Release $(NEW_VERSION)"
	git push origin $(NEW_VERSION)

release-patch: ## Create a new patch release (v1.0.0 -> v1.0.1)
	@echo "Creating patch release $(NEXT_VERSION)"
	git tag -a $(NEXT_VERSION) -m "Release $(NEXT_VERSION)"
	git push origin $(NEXT_VERSION)

release-check: ## Check release configuration
	goreleaser check

release-local: ## Test release locally
	goreleaser release --snapshot --clean --skip-publish

ci-release: ## Run release process (used by CI)
	goreleaser release --clean

# Dev targets
dev-deps: ## Install development dependencies
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/goreleaser/goreleaser@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

uninstall-deps: ## Uninstall development dependencies
	go clean -i github.com/golangci/golangci-lint/cmd/golangci-lint
	go clean -i github.com/goreleaser/goreleaser
	go clean -i github.com/securego/gosec/v2/cmd/gosec

# Default target when just running 'make'
all: lint security test build ## Run lint, security, test, and build