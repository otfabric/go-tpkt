SHELL := /bin/bash

GO ?= go
PKGS := ./...

.PHONY: help test test-race vet lint fmt fuzz fuzz-decode fuzz-parse bench tidy check clean coverage coverage-html coverage-clean

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run unit tests
	@echo "Running unit tests..."
	$(GO) test $(PKGS)

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	$(GO) test -race $(PKGS)

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet $(PKGS)

lint: ## Run staticcheck
	@echo "Running staticcheck"
	@staticcheck $(PKGS)

lint-ci: ## Run golangci-lint
	@echo "Running golangci-lint"
	@golangci-lint run $(PKGS)

fmt: ## Run go fmt
	@echo "Running go fmt..."
	@gofmt -w .

coverage: ## Run tests with coverage profile and text summary
	@echo "Running coverage..."
	$(GO) test -coverprofile=coverage.out $(PKGS)
	$(GO) tool cover -func=coverage.out | tee coverage.txt

coverage-html: coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html

coverage-clean: ## Remove coverage artifacts
	@echo "Removing coverage artifacts..."
	rm -f coverage.out coverage.txt coverage.html

fuzz: ## Run short fuzz tests
	@echo "Running fuzz tests..."
	$(GO) test -fuzz=FuzzDecode -fuzztime=5s $(PKGS)
	$(GO) test -fuzz=FuzzParse -fuzztime=5s $(PKGS)

fuzz-decode: ## Run Decode fuzz target
	@echo "Running Decode fuzz target..."
	$(GO) test -fuzz=FuzzDecode -fuzztime=10s $(PKGS)

fuzz-parse: ## Run Parse fuzz target
	@echo "Running Parse fuzz target..."
	$(GO) test -fuzz=FuzzParse -fuzztime=10s $(PKGS)

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -run=^$$ -bench=. -benchmem $(PKGS)

tidy: ## Tidy module files
	@echo "Tidying module files..."
	$(GO) mod tidy

check: fmt tidy vet lint lint-ci test test-race coverage ## Run core release checks

clean: ## Clean test cache
	@echo "Cleaning test cache..."
	$(GO) clean -testcache