# Testing

All tests are in the root package. They need Go 1.26 or later and no external service; the TCP tests listen on a `127.0.0.1` port chosen by the system. No test is skipped on a missing prerequisite, so there are no `*_REQUIRE_*` variables.

## Tiers

| Tier | Command | Contents |
|---|---|---|
| Unit | `make test` | Codec, reader and writer tests, including error paths |
| Race | `make test-race` | The same tests with the race detector, including the full-duplex tests |
| Stream | part of `make test` | `pipe_test.go` over `net.Pipe`, `tcp_test.go` over loopback TCP |
| Examples | part of `make test` | `example_test.go` |
| Fuzz | `make fuzz` | Every Fuzz target for `FUZZTIME` (default 5s) |
| Single fuzz target | `make fuzz-decode`, `make fuzz-roundtrip`, `make fuzz-reader`, `make fuzz-reserved` | One target for 10 s |
| Benchmarks | `make bench` | `bench_test.go` |
| Coverage | `make coverage`, `make coverage-check`, `make coverage-pkg-check`, `make coverage-html` | Profile, total gate, per-package gate, HTML report |
| Cross-compile | `make cross` | Builds for the platforms in the Makefile; runs no tests |

There is no end-to-end tier and no interop tier: the module has no peer to run against ([LIMITATIONS.md](LIMITATIONS.md)). `make check` runs the local gate, coverage gates included.

## Boundary, limit and stream tests

The cases are explicit tests, not only fuzz seeds, so that a bound that moves by one octet fails a named case.

| Subject | Tests |
|---|---|
| Declared length 0, 1, 4, 6, 7, 8, 65534 and 65535, through `DecodePacket` and `ReadPacket` | `TestDeclaredLengthBoundaries` |
| Round trip of the smallest and the largest payload and their neighbours, through both pairs of entry points | `TestRoundTripPayloadBoundaries` |
| `ReaderConfig.MaxPacketLength` at both ends of its range, at zero and at the limits of `int` | `TestNewReaderMaxPacketLengthBoundaries` |
| A packet at the receive limit and one octet above it, for limits 7, 12, 4096, 65534 and 65535 | `TestReaderLimitBoundaries` |
| Allocation: a header above the limit is refused within 2 KiB for each call; a header at the limit allocates its payload and nothing that grows beyond it | `TestReaderOversizeAllocationBudget`, `TestReaderAllocationBoundedByLimit` |
| End of the stream after every prefix of a packet, with a plain and with a wrapped EOF | `TestReaderEOFClassification` |
| An error of the stream keeps its identity in every phase of a packet | `TestReaderStreamErrorKeepsIdentity` |
| A `Reader` or `Writer` stays usable after an error on a packet boundary: a deadline, an EOF, a `Write` that accepts nothing | `TestReaderUsableAfterErrorAtPacketBoundary`, `TestWriterUsableAfterErrorBeforeAnyOctet` |
| After an error inside a packet, every later call returns `ErrStreamMisaligned` with the cause and does not touch the stream | `TestReaderStaysFailedAfterStreamErrorInsidePacket`, `TestReaderRetryInsidePacketReturnsNoPayload`, `TestReaderStaysFailedAfterRefusedOrTruncatedPacket`, `TestWriterStaysFailedAfterErrorInsidePacket` |
| The capacity of the `DecodePacket` result ends at the packet | `TestDecodePacketResultEndsAtPacket` |
| A zero-value or nil `Reader` or `Writer` returns its sentinel | `TestReaderZeroValueAndNil`, `TestWriterZeroValueAndNil` |
| Arbitrary octets, receive limit and chunking, with one stream error at an arbitrary offset, compared with a reference walk over the same octets | `FuzzReaderStream` |

The two allocation tests measure `runtime.MemStats.TotalAlloc`, which counts the whole process: they rely on the tests of this package not running in parallel.

## Lifecycle contract matrix

`Reader` and `Writer` are the only types. They start no goroutine, hold no lock, timer, queue or callback, keep no state besides the error of a call that failed inside a packet, and do not own the stream ([ARCHITECTURE.md](ARCHITECTURE.md#goroutines-and-ownership)). No row of the lifecycle contract matrix (L1 to L16 in otfabric/engineering `standards/lifecycle.md`) applies to them as a Close contract. Two rows have an analogue in the caller's stream, and L14 applies to the constructors.

| Row | Covered by, or the reason it does not apply | Gap |
|---|---|---|
| L1 Close twice | n/a: neither type has `Close`; the caller closes the stream | — |
| L2 Two goroutines call Close | n/a: no `Close` | — |
| L3 Close during a blocked read | n/a as a Close contract. Analogue: the error of the stream is returned wrapped, in the header phase (`TestReaderZeroByteCustomError`) and in every phase of a packet (`TestReaderStreamErrorKeepsIdentity`) | No test closes a `net.Pipe` under a blocked `ReadPacket` |
| L4 Close during a write | n/a as a Close contract. Analogue: `TestWriterShortWriteThenError` asserts the error of a partial write, and `TestWriterStaysFailedAfterErrorInsidePacket` that no packet follows an incomplete one on the stream | — |
| L5 Close during handshake | n/a: no handshake | — |
| L6 Cancel a pending request | n/a: no request and response; no context | — |
| L7 Response after timeout | n/a: no correlation | — |
| L8 Response after reconnect | n/a: no reconnect | — |
| L9 Callback calls a public method | n/a: no callbacks | — |
| L10 Callback calls Close | n/a: no callbacks | — |
| L11 Slow callback | n/a: no callbacks | — |
| L12 Queue at capacity | n/a: no queue or buffer between packets | — |
| L13 Concurrent sessions | n/a: no sessions. One `Reader` and one `Writer` on a connection: `TestPipeFullDuplex` and `TestTCPFullDuplexSmoke`, under the race detector | — |
| L14 Failure during construction | `TestNewReaderValidation`, `TestNewReaderMaxPacketLengthBoundaries`, `TestNewWriterNil`: the constructors acquire nothing | — |
| L15 Shutdown(ctx) | n/a: no shutdown | — |
| L16 Close racing reconnect | n/a: no reconnect | — |

## CI

`.github/workflows/ci.yml` calls the shared `go-ci` workflow: tests and race tests on Go 1.26 and 1.27 on Linux, the coverage gate, govulncheck, and a smoke run of 10 s for each Fuzz target. The `package-coverage` job runs `make coverage-pkg-check` (`COVER_PKG_MIN`, 95%).

`.github/workflows/nightly.yml` runs every Fuzz target for 5 minutes and uploads a failing input as an artifact.

## Coverage policy

The floor is 90%, set by `coverage-min` in `ci.yml` and `COVER_MIN` in the Makefile; change both together. Coverage is measured over the whole module and nothing is excluded.

## Fuzz regressions

When a fuzz run fails, Go writes the input to `testdata/fuzz/<FuzzTarget>/`. Commit that file with the fix: `go test` replays it as a regression case. The module has no such corpus files at present; seeds are added with `f.Add` in `fuzz_test.go`.

## When CI runs

| Event | Workflow | What runs |
|---|---|---|
| Pull request | `pr.yml` | `docs.yml` (docsync, otfabric-policy) always; `ci.yml` unless every changed file is documentation; then `OT Fabric Required Checks` |
| Push to a branch | none | nothing; dispatch `ci.yml` to test a branch without a pull request |
| Push to `main` | `docs.yml` | the light checks; the merged commit was tested as the pull request's merge commit |
| Schedule | `nightly.yml` | the scheduled runs, on `main` |
| Release | `release.yml` | the release workflow tests the release commit again, and builds, vets and tests go-cotp against it before it creates the tag |

Documentation means an added or modified `*.md` file outside `.github/`, `testdata/` and `vendor/`, other than `AGENTS.md` and `CLAUDE.md`. A deleted or renamed document, or any other file, runs everything. `OT Fabric Required Checks` fails unless every job that applies succeeded; its summary lists them.
