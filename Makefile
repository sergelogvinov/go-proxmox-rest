
SHA ?= $(shell git describe --match=none --always --abbrev=7 --dirty)
TAG ?= $(shell git describe --tag --always --match v[0-9]\*)

############

# Help Menu

define HELP_MENU_HEADER
# Getting Started

To build this project, you must have the following installed:

- git
- make
- golang 1.26+
- golangci-lint

endef

export HELP_MENU_HEADER

help: ## This help menu
	@echo "$$HELP_MENU_HEADER"
	@grep -E '^[a-zA-Z0-9%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

############

.PHONY: schema
schema:
	wget https://raw.githubusercontent.com/proxmox/proxmox-rs/refs/heads/master/pve-api-types/pve-api.json -O docs/pve-api.json

############
#
# Build Abstractions
#

.PHONY: clean
clean: ## Clean
	rm -rf bin/ dist/ .cache/ .gocache/ vendor/
	rm -rf mimiops-mcp_*.mcpb

.PHONY: tools
tools:
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.21.0
	go install github.com/google/go-licenses@latest

.PHONY: lint
lint: ## Lint Code
	golangci-lint run --config .golangci.yml

.PHONY: vet
vet: ## Vet Code
	go vet ./...

.PHONY: unit
unit: ## Unit Tests
	go test -tags=unit $(shell go list ./...) $(TESTARGS)

.PHONY: e2e
e2e: ## End-to-End Tests (requires PVE_E2E_* env vars, see docs/e2e.md)
	go test -tags=e2e ./tests/e2e/... -v $(TESTARGS)

e2e-%: ## End-to-End Tests for specific module only
	go test -tags=e2e ./tests/e2e/$(subst -,/,$(*$))/... -v $(TESTARGS)

.PHONY: test
test: lint unit ## Run all tests

.PHONY: licenses
licenses:
	go-licenses check ./... --disallowed_types=forbidden,restricted,unknown

.PHONY: conformance
conformance: ## Conformance
	docker run --rm -it -v $(PWD):/src -w /src ghcr.io/siderolabs/conform:v0.1.0-alpha.31 enforce
