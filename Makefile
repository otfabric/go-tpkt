SHELL := /bin/bash

# Prefer module-local dependency resolution when a sibling go.work exists.
export GOWORK := off

PKGS := ./...

.PHONY: help test test-race vet lint lint-ci fmt fuzz fuzz-decode fuzz-roundtrip fuzz-reader fuzz-reserved bench tidy vuln check clean coverage coverage-html coverage-clean

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run unit tests
	@echo "Running unit tests..."
	@go test $(PKGS)

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@go test -race $(PKGS)

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet $(PKGS)

lint: ## Run staticcheck
	@echo "Running staticcheck"
	@staticcheck $(PKGS)

lint-ci: ## Run golangci-lint
	@echo "Running golangci-lint"
	@golangci-lint run $(PKGS)

fmt: ## Run go fmt
	@echo "Running gofmt"
	@gofmt -w .
	@echo "Running go fmt"
	@go fmt $(PKGS)

coverage: ## Run tests with coverage profile and text summary
	@echo "Running coverage..."
	@go test -coverprofile=coverage.out $(PKGS)
	@go tool cover -func=coverage.out | tee coverage.txt

coverage-html: coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html

coverage-clean: ## Remove coverage artifacts
	@echo "Removing coverage artifacts..."
	rm -f coverage.out coverage.txt coverage.html

fuzz: ## Run short fuzz tests (all targets)
	@echo "Running fuzz tests..."
	@go test -fuzz=FuzzDecodePacket -fuzztime=5s .
	@go test -fuzz=FuzzEncodeDecodePacket -fuzztime=5s .
	@go test -fuzz=FuzzReaderChunking -fuzztime=5s .
	@go test -fuzz=FuzzReservedPolicy -fuzztime=5s .

fuzz-decode: ## Run FuzzDecodePacket
	@echo "Running FuzzDecodePacket..."
	@go test -fuzz=FuzzDecodePacket -fuzztime=10s .

fuzz-roundtrip: ## Run FuzzEncodeDecodePacket
	@echo "Running FuzzEncodeDecodePacket..."
	@go test -fuzz=FuzzEncodeDecodePacket -fuzztime=10s .

fuzz-reader: ## Run FuzzReaderChunking
	@echo "Running FuzzReaderChunking..."
	@go test -fuzz=FuzzReaderChunking -fuzztime=10s .

fuzz-reserved: ## Run FuzzReservedPolicy
	@echo "Running FuzzReservedPolicy..."
	@go test -fuzz=FuzzReservedPolicy -fuzztime=10s .

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -run=^$$ -bench=. -benchmem $(PKGS)

tidy: ## Tidy module files
	@echo "Tidying module files..."
	@go mod tidy

vuln: ## Run govulncheck (install: go install golang.org/x/vuln/cmd/govulncheck@latest)
	@echo "Running govulncheck"
	@govulncheck $(PKGS)

check: fmt tidy vet lint lint-ci vuln test test-race coverage ## Run core release checks

clean: ## Clean test cache
	@echo "Cleaning test cache..."
	@go clean -testcache
