# Contributing to otfabric/go-tpkt

Thank you for your interest in contributing. This document explains how to get started.

## Development setup

- **Go**: 1.23 or later.

```sh
git clone https://github.com/otfabric/go-tpkt.git
cd go-tpkt
go mod download
```

## Running tests

- **Unit tests**: `make test` (runs `go test ./...`).
- **Race tests**: `go test -race ./...` or rely on CI, which runs race tests on Go 1.23.
- **Benchmarks**: `make bench` to run the small benchmark suite.

## Code style and linting

- Format code: run `gofmt` (or your editor’s “format on save”) on modified files.
- Lint: `golangci-lint run` (config in `.golangci.yml`) and `go vet ./...`.

Please run `go test ./...` and `golangci-lint run` before submitting a PR.

## Submitting changes

1. Open an issue or pick an existing one to discuss the change.
2. Fork the repo, create a branch, and make your changes.
3. Add or update tests as needed.
4. Run `make test` and `golangci-lint run`.
5. Open a pull request with a clear description and reference to the issue.

## Error handling

Prefer sentinel errors from `errors.go` and wrap with `%w` so callers can use `errors.Is` / `errors.As`. Avoid string-only error comparisons; always return typed/sentinel errors wrapped with context.

## Documentation

When you change **public API signatures** (function parameters, return types, or exported types), update:

- **[doc.go](doc.go)** — keep package documentation in sync.
- **[README.md](README.md)** — if it references the changed API or examples.
- **[RELEASE.md](RELEASE.md)** — add a short note if the change is user-visible.

Also keep doc comments on exported symbols in sync with behavior.

