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
hooks: ## Arm the commit-msg and pre-commit guards (.githooks/) for this clone
	git config core.hooksPath .githooks

# Wraps scripts/install.sh, which is also the curl | sh front door (README.md § Install), so
# the arch resolution, temp dir, checksum verification and PATH warnings exist in one place.
.PHONY: install
install: ## Install the latest released kb into ~/.local/bin (KB_BIN_DIR, KB_VERSION override)
	@sh scripts/install.sh

.PHONY: release-check
release-check: ## Validate .goreleaser.yaml and build a local snapshot release into ./dist
	goreleaser check
	goreleaser release --snapshot --clean
