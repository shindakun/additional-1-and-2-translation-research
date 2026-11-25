.PHONY: all help build run test race cover vet fmt lint md-lint tidy tidy-check hooks check clean

all: check ## Default: run the full local check suite

help: ## Show available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

BIN     := kfa
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build: ## Build every package (no binary written)
	go build -o /dev/null ./...

$(BIN): ## Build the binary for this host
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/$(BIN)

run: $(BIN) ## Build and run (pass args with ARGS=...)
	./$(BIN) $(ARGS)

# --- Checks ----------------------------------------------------------------

test: ## Run the test suite
	go test ./...

race: ## Run the test suite under the race detector
	go test -race ./...

cover: ## Run tests with a coverage profile
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

vet: ## go vet
	go vet ./...

fmt: ## Format Go sources
	# "." rather than a list of directories: a project that drops internal/ or
	# cmd/ (no database, or a library with no binary) would otherwise fail on a
	# path that is not there. gofmt skips testdata/ and dot-directories itself.
	gofmt -w .

lint: ## golangci-lint (config: .golangci.yml)
	golangci-lint run

md-lint: ## markdownlint (config: .markdownlint-cli2.jsonc)
	npx --yes markdownlint-cli2 "**/*.md"

tidy: ## Tidy the module file
	go mod tidy

tidy-check: ## Fail if go.mod/go.sum are not tidy (same check CI runs)
	# Note: this rewrites go.mod and go.sum in place before diffing them, so a
	# failure here has already fixed the file. Re-run `make check` and commit
	# the result. CI does exactly this, so a pass here means a pass there.
	#
	# The diff needs a git repo with go.mod committed. A freshly generated
	# project has neither, so tidy still runs and the comparison is skipped
	# rather than failing on something the user has not had a chance to do yet.
	#
	# go.sum is added to the diff only when it exists: a project with no
	# external dependencies never has one, and `git diff` on a path that is not
	# in the working tree exits 128 rather than reporting no changes.
	go mod tidy
	@if git rev-parse --git-dir >/dev/null 2>&1 && git ls-files --error-unmatch go.mod >/dev/null 2>&1; then \
	  set -- go.mod; \
	  [ -f go.sum ] && set -- "$$@" go.sum; \
	  git diff --exit-code -- "$$@"; \
	else \
	  echo "tidy-check: not a git repo with go.mod tracked yet; ran go mod tidy, skipped the diff"; \
	fi

hooks: ## Install pre-commit hooks
	pre-commit install

check: fmt vet lint test md-lint tidy-check ## Run the full local check suite

clean: ## Remove build output and coverage data
	rm -f $(BIN) coverage.out coverage.html
