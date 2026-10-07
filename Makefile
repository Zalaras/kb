.DEFAULT_GOAL := help

.PHONY: help
help: ## List targets
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the kb binary into ./bin/kb
	go build -o bin/kb ./cmd/kb

.PHONY: test
test: ## Run the tests, uncached
	go test -count=1 ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format Go sources
	golangci-lint fmt

.PHONY: tidy
tidy: ## Tidy go.mod/go.sum
	go mod tidy

.PHONY: gen-kb
gen-kb: ## Regenerate this repo's own generated knowledge files (docs/INDEX.md)
	go run ./cmd/kb gen

.PHONY: check-kb
check-kb: ## Check this repo's own records and generated files
	go run ./cmd/kb check

.PHONY: check
check: lint test check-kb ## Lint + test + check-kb

.PHONY: hooks
hooks: ## Arm the pre-commit identity guard (.githooks/) for this clone
	git config core.hooksPath .githooks
