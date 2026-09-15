.DEFAULT_GOAL := help

## Frontend tooling
NPM ?= npm
NPX ?= npx
NODE_MODULES := node_modules/.package-lock.json

## Tool Versions
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2

# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.7.0

## Shared development tools
include bin/dev-tools.mk
include $(call dev-tools-module,tag)
include $(call dev-tools-module,help)

## Project-local tools
GOLANGCI_LINT := bin/golangci-lint

## Build Configuration
BINARY ?= bin/kumbuka-plugin
COMMAND ?= ./cmd/kumbuka-plugin
BUILD_VERSION ?= dev
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS ?= -s -w -X main.Version=$(BUILD_VERSION) -X main.Commit=$(BUILD_COMMIT)

## Formatting
PRETTIER_MD_SOURCES := \
	README.md \
	WIRE.md \
	"markdown/**/*.md" \
	"pluginpackage/**/*.md"


##@ Development

.PHONY: download
download: $(NODE_MODULES) dev-tools ## Download Go, formatting, and development dependencies.
	go mod download

.PHONY: build
build: ## Build the Kumbuka plugin development CLI.
	@mkdir -p $(dir $(BINARY))
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) $(COMMAND)

.PHONY: vet
vet: ## Run Go static analysis.
	go vet ./...

.PHONY: test
test: vet ## Run unit tests.
	go test -covermode=set -timeout=3m ./...

.PHONY: test-fresh
test-fresh: vet ## Run unit tests without the Go test cache.
	go test -covermode=set -count=1 -timeout=3m ./...

.PHONY: test-race
test-race: vet ## Run unit tests with the race detector.
	go test -race -count=1 -timeout=3m ./...

.PHONY: cover
cover: ## Display Go test coverage.
	go test -coverprofile=coverage.out -covermode=set -count=1 -timeout=3m ./...
	go tool cover -html=coverage.out

.PHONY: clean
clean: ## Clean generated files.
	rm -f $(BINARY) coverage.out coverage.html


##@ Formatting

.PHONY: fmt
fmt: fmt-go fmt-md ## Format all supported files.

.PHONY: fmt-go
fmt-go: ## Format Go code.
	go fmt ./...

.PHONY: fmt-md
fmt-md: $(NODE_MODULES) ## Format Markdown files.
	$(NPX) prettier --write $(PRETTIER_MD_SOURCES)

.PHONY: check-md
check-md: $(NODE_MODULES) ## Check Markdown formatting.
	$(NPX) prettier --check $(PRETTIER_MD_SOURCES)


##@ Linting

.PHONY: lint
lint: check-md lint-go ## Run all linters and formatting checks.

.PHONY: lint-go
lint-go: golangci-lint ## Run golangci-lint.
	$(call run-tool,$(GOLANGCI_LINT),run)

.PHONY: lint-fix
lint-fix: fmt-md golangci-lint ## Run linters and apply fixes.
	$(call run-tool,$(GOLANGCI_LINT),run --fix)


##@ Dependencies

$(NODE_MODULES): package.json package-lock.json
	$(NPM) ci

.PHONY: dev-tools
dev-tools: $(DEV_TAG) $(MAKE_HELP) $(GO_INSTALL_TOOL) ## Download the pinned development tools.

.PHONY: golangci-lint
golangci-lint: $(GO_INSTALL_TOOL) ## Download golangci-lint locally if necessary.
	@$(GO_INSTALL_TOOL) \
		--target "$(GOLANGCI_LINT)" \
		--package github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		--tool-version "$(GOLANGCI_LINT_VERSION)"
