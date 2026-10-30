# Testing

All tests are in the root package. They need Go 1.23 or later and no external service; the TCP tests listen on a `127.0.0.1` port chosen by the system. No test is skipped on a missing prerequisite, so there are no `*_REQUIRE_*` variables.

## Tiers

| Tier | Command | Contents |
|---|---|---|
| Unit | `make test` | Codec, reader and writer tests, including error paths |
| Race | `make test-race` | The same tests with the race detector, including the full-duplex tests |
| Stream | part of `make test` | `pipe_test.go` over `net.Pipe`, `tcp_test.go` over loopback TCP |
| Examples | part of `make test` | `example_test.go` |
| Fuzz | `make fuzz` | Every Fuzz target for 5 s |
| Single fuzz target | `make fuzz-decode`, `make fuzz-roundtrip`, `make fuzz-reader`, `make fuzz-reserved` | One target for 10 s |
| Benchmarks | `make bench` | `bench_test.go` |
| Coverage | `make coverage`, `make coverage-check`, `make coverage-html` | Profile, gate, HTML report |
| Cross-compile | `make cross` | Builds for the platforms in the Makefile; runs no tests |

There is no end-to-end tier and no interop tier: the module has no peer to run against ([LIMITATIONS.md](LIMITATIONS.md)). `make check` runs the local gate; it uses `make coverage`, which reports without enforcing the floor, so run `make coverage-check` as well.

## CI

`.github/workflows/ci.yml` calls the shared `go-ci` workflow: tests and race tests on Go 1.23 to 1.26 on Linux, the coverage gate, govulncheck, and a smoke run of 10 s for each Fuzz target.

## Coverage policy

The floor is 90%, set by `coverage-min` in `ci.yml` and `COVER_MIN` in the Makefile; change both together. Coverage is measured over the whole module and nothing is excluded.

## Fuzz regressions

When a fuzz run fails, Go writes the input to `testdata/fuzz/<FuzzTarget>/`. Commit that file with the fix: `go test` replays it as a regression case. The module has no such corpus files at present; seeds are added with `f.Add` in `fuzz_test.go`.
