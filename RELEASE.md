# go-tpkt Releases

## v1.2.0

**Date:** 2026-11-14
**Previous release:** v1.1.0

### Breaking

- **A `Reader` or a `Writer` that failed inside a packet stays failed.** When `ReadPacket` has consumed, or `WritePacket` has written, one or more octets of a packet and the call then returns an error, every later call on the same `Reader` or `Writer` returns an error that matches the new `ErrStreamMisaligned` and wraps the first error. It reads and writes nothing. This holds for every error inside a packet: a refused header, an end of the stream, a deadline or another error of the stream, a `Write` that accepts nothing. The call that fails returns its error as before, and `errors.Is` and `errors.As` find the cause in the later errors too.
- **Why:** such an error leaves the stream inside a packet, and a later call went on from there. A `ReadPacket` repeated after a read deadline took payload octets for a header and could return a payload that was never sent as a packet, without an error. A `WritePacket` behind an incomplete packet made the peer read two packets as one. Only a refused header was documented as the end of the stream.
- **What stays as it was:** an error before any octet of a packet moved leaves the `Reader` or `Writer` usable. That is `io.EOF`, a deadline on an idle connection, and a `Write` that fails or accepts nothing at the start of a packet. A caller that repeats the call after those, as go-cotp does after a read deadline between packets, works as before. `WritePacket` reports a payload-size error first in every state.
- **No option restores the old behaviour.** Going on with a stream that is inside a packet was never safe, so there is nothing to opt back into.
- **What a caller does instead:** close the stream after the first error inside a packet, or at the latest when a call returns `ErrStreamMisaligned`, and open a new connection with a new `Reader` and `Writer`. A caller that already closed the connection after every error other than `io.EOF` needs no change.

### Added

- `ErrStreamMisaligned`, the sentinel of the failed state. No other exported identifier is added, removed or changed.

### Fixed

- `ReadPacket` reports a stream that ends inside a header as `io.ErrUnexpectedEOF` also when the stream wraps its EOF in an error of its own. Such an error no longer matches `io.EOF`, which the documented contract keeps for a packet boundary. A wrapped EOF on a packet boundary is still returned as `io.EOF`.
- The payload that `DecodePacket` returns has no capacity beyond the packet. Appending to it copies, and the rest of the caller's buffer stays unchanged. The payload still aliases `packet`.
- `ReadPacket` on a zero-value or nil `Reader` returns `ErrNilReader`, and `WritePacket` on a zero-value or nil `Writer` returns `ErrNilWriter`. The type comments say that `NewReader` and `NewWriter` construct the values.

### Documentation

- The doc comments of `Reader`, `Writer`, `ReadPacket` and `WritePacket`, API.md ("State of the stream after an error"), LIMITATIONS.md and ARCHITECTURE.md describe the failed state: which errors leave a `Reader` or `Writer` usable, which do not, and what the later calls return.
- ERRORS.md has `ErrStreamMisaligned`, describes the errors of the stream API that are not sentinels of this package (`io.EOF`, `io.ErrUnexpectedEOF`, `io.ErrShortWrite`, the wrapped errors of the stream), and says that `ErrInvalidPacketLength` is about the length a header declares.
- SECURITY.md has a threat model and a table of resource limits: what one `ReadPacket` call allocates for a peer's header, and that `ReaderConfig.MaxPacketLength` sets the limit.
- TESTING.md lists the boundary, limit and stream tests and the lifecycle contract matrix (L1 to L16) with the reason for each row that does not apply.
- The install command in the README no longer names a version.
- `spec/README.md` and `spec/SHA256SUMS` record the source, the comparison with the RFC Editor's texts and the checksums of the two RFC texts.

### Tests

- The failed state: an error after each number of octets of a header and of a payload, a refused header and an end of the stream inside a packet, for `Reader`; an error with and behind each number of octets of a packet and a `Write` that accepts nothing, for `Writer`; and the errors on a packet boundary after which the call is repeated.
- `FuzzReaderStream` gives arbitrary octets, receive limits and chunkings to `ReadPacket`, with one stream error at an arbitrary offset, and compares each call with a reference walk over the input.
- Boundary tables for the declared length, the payload size, `ReaderConfig.MaxPacketLength` and the receive limit, and an allocation budget for a header above and at the limit.
- `TestPipeAdversarialBadVersion` ends when the reading side closes the connection, and checks the error of the pending write.
- The benchmarks report allocations, check their results and run for payloads of 3, 1024 and 65531 octets. `BenchmarkEncodePacket`, `BenchmarkDecodePacket`, `BenchmarkReadPacket` and `BenchmarkWritePacket` replace `BenchmarkEncodePacketSmall`, `BenchmarkDecodePacketSmall` and `BenchmarkReadWritePacket`.

### CI

- The release workflow builds, vets and tests go-cotp against the release commit before it creates the tag (`consumers`).

---

## v1.1.0

**Date:** 2026-11-03
**Previous release:** v1.0.3

### Changed

- **Go 1.26 is now the minimum** (`go 1.26` in `go.mod`, was 1.23). CI tests Go 1.26 and 1.27. Modules that import go-tpkt v1.1.0 need Go 1.26 or newer; the API and the wire behaviour are unchanged.
- `go fix ./...` applied: tests use `min` and `sync.WaitGroup.Go`.

---

## v1.0.3

**Date:** 2026-11-03
**Previous release:** v1.0.2

### Changed

- `EncodePacket` checks the total packet length itself against `[MinPacketLength, MaxPacketLength]` before writing it as a 16-bit length, so a static check can see that the value fits (gosec G115). The accepted payload sizes and the errors are unchanged.

---

## v1.0.2

**Date:** 2026-10-31
**Previous release:** v1.0.1

### Changed

- Policy 1.2: build output in `bin/` and `release/`, a per-package coverage floor, and more tests and CI. No change in behaviour.

---

## v1.0.1

**Date:** 2026-07-30
**Previous release:** v1.0.0

### Summary

Docs and Makefile polish only. No public API surface changes.

### Changed

- **API.md** — added an [Ownership](API.md#ownership) subsection documenting
  buffer aliasing for `DecodePacket` versus new allocations from
  `EncodePacket` / `ReadPacket` / `WritePacket`.
- **README** — repository structure overview (flat `tpkt` package layout).
- **Makefile** — exports `GOWORK=off`; adds `make vuln` (`govulncheck`) to
  `check`; fuzz targets aligned to real names (`FuzzDecodePacket`,
  `FuzzEncodeDecodePacket`, `FuzzReaderChunking`, `FuzzReservedPolicy`).
- **Tests** — cover `decodeHeader` short-buffer and empty `readFullPayload`
  guards; drop unreachable partial-header EOF remap in `readFullHeader`
  (`io.ReadFull` already returns `ErrUnexpectedEOF`). Statement coverage
  **100%**.

Import path remains `github.com/otfabric/go-tpkt`.

---

## v1.0.0

First stable API release.

This release introduces a deliberately breaking cleanup from the v0 frame
terminology to consistent packet / TPKT terminology. No compatibility aliases
are provided.

### Changed

- Replaced the buffer API with:
  - `EncodePacket`
  - `DecodePacket`
- Replaced the stream API with:
  - `ReadPacket`
  - `WritePacket`
- Added:
  - `NewReader(io.Reader, ReaderConfig) (*Reader, error)`
  - `NewWriter(io.Writer) (*Writer, error)`
- `ReaderConfig.MaxPacketLength` represents the total TPKT length, including
  the four-byte header.
- A zero `MaxPacketLength` (including `ReaderConfig{}`) selects the default
  protocol maximum (`MaxPacketLength`).
- Invalid configured maximum lengths are rejected rather than silently clamped.
- Nil readers and writers are rejected during construction.
- The reserved header octet is ignored on input, as recommended by
  RFC 2126 §6.10.
- Encoded packets always contain zero in the reserved octet.
- `DecodePacket` accepts exactly one complete TPKT and rejects truncated
  input, length mismatches, and trailing bytes.
- `ReadPacket` returns `io.EOF` only when the stream ends cleanly before any
  byte of the next TPKT is consumed.
- EOF after a partial header, after an accepted header, or during the payload
  is reported as `io.ErrUnexpectedEOF`.
- Added exported packet and payload bounds:
  - `MinPacketLength`
  - `MaxPacketLength`
  - `MinPayloadLength`
  - `MaxPayloadLength`
- Renamed and simplified the exported error model. Exported sentinels:
  - `ErrNilReader`
  - `ErrNilWriter`
  - `ErrTooShort`
  - `ErrInvalidVersion`
  - `ErrInvalidPacketLength`
  - `ErrInvalidMaxPacketLength`
  - `ErrLengthMismatch`
  - `ErrPacketTooLarge`
  - `ErrPayloadTooShort`
  - `ErrPayloadTooLarge`
- Standardized exported documentation and examples on packet, TPKT, header,
  and payload / TPDU terminology.

### Removed

- `Frame`
- `Parse`
- `Encode`
- `Decode`
- `ReadFrame`
- `WriteFrame`
- `ReaderOption`
- `WithMaxFrameSize`
- `ErrInvalidReserved`
- `ErrFrameTooLarge`
- `ErrInvalidLength`

No deprecated wrappers or compatibility aliases are retained.

### Stream behavior

- Each `ReadPacket` call returns exactly one TPKT payload.
- Additional packets already available in the underlying stream remain
  available for subsequent calls.
- After an invalid header or oversized-packet rejection, the stream is
  considered unusable because the implementation does not drain
  attacker-controlled packet data. The caller should close the connection.
- One `Reader` and one `Writer` may operate concurrently over the same
  full-duplex connection.
- An individual `Reader` or `Writer` must not be used concurrently by
  multiple goroutines.

### Documentation and tests

- Rewrote [API.md](API.md) for the v1 API.
- Added a TPKT requirements matrix distinguishing:
  - RFC normative requirements
  - RFC recommendations
  - derived constraints
  - security invariants
  - library API contracts
- Updated [README.md](README.md) getting-started examples to use the packet API.
- Aligned [CONTRIBUTING.md](CONTRIBUTING.md) with Go 1.23 and later.
- Expanded deterministic stream-contract coverage for:
  - fragmented headers and payloads
  - coalesced packets
  - readers returning both bytes and `io.EOF`
  - zero-byte custom reader errors
  - truncated headers and payloads
  - short writes
  - zero-progress writes
  - partial-write failures
  - configured packet-length limits
- Added full-duplex `net.Pipe` tests, adversarial peer tests, and localhost
  TCP smoke tests.
- Added fuzz targets covering:
  - packet decoding
  - encode/decode round trips
  - fragmented reads
  - reserved-octet receive behavior

### Requirements

- Go 1.23 or later.

### Standards scope

This release implements TPKT framing according to:

- RFC 1006 §6
- RFC 2126 §4.3
- RFC 2126 §6.10

It does not implement the broader RFC 2126 ITOT transport profile, including
COTP Class 0 or Class 2 behavior, expedited-data connections, transport
addressing, or connection-management procedures.

TPDU contents remain opaque to this package. `go-tpkt` enforces the TPKT
packet-length constraints but does not validate COTP structure or semantics.

### Migration

Downstream consumers, including go-cotp, must migrate to the v1 packet API and
renamed errors.

There are no compatibility aliases.

---

## v0.1.3

**Changed**: Open-source release preparation and documentation improvements. No API or behavior changes.

- Added `// SPDX-License-Identifier: MIT` headers to all first-party Go source files.
- Expanded [API.md](API.md) into a full public API reference (wire format, types, functions, errors, streaming semantics, ownership, and usage patterns).
- Updated [README.md](README.md): standardized badge block (pkg.go.dev, tokenless Codecov), table of contents, runnable getting-started examples, license section, and links to API.md for detailed usage.

---

## v0.1.2

**Changed**: Increased minimum required Go version to 1.23 (was 1.21). All documentation, CI, and go.mod references updated accordingly. No code changes.

This release ensures compatibility with Go 1.23. No new features or bugfixes are included.

---

## v0.1.1

**Changed**: Lowered minimum required Go version to 1.21 (was 1.22). All documentation, CI, and go.mod references updated accordingly. No code changes.

This release ensures compatibility with Go 1.21. No new features or bugfixes are included.

---

## v0.1.0

Initial public release of `github.com/otfabric/go-tpkt`, a small, idiomatic Go library that implements TPKT framing as defined in RFC 1006.

### Highlights

- **TPKT codec**
  - `Frame` type with `Payload []byte`, `Len`, and `MarshalBinary`.
  - `Encode`, `Decode`, and `Parse` helpers with strict RFC 1006 validation (version, reserved byte, length, and buffer consistency).
  - Exported size constants: `MinPacketLength` (7) and `MaxPacketLength` (65535).
  - Clear, structured error model with sentinel errors (`ErrTooShort`, `ErrInvalidVersion`, `ErrInvalidReserved`, `ErrInvalidLength`, `ErrLengthMismatch`, `ErrFrameTooLarge`).

- **Streaming API**
  - `Reader` over `io.Reader` with `ReadFrame` and configurable `WithMaxFrameSize`, enforcing max frame size before allocation.
  - `Writer` over `io.Writer` with `WriteFrame`, handling short writes and wrapped underlying errors correctly.

- **Safety and correctness**
  - Strict minimum TPKT packet length of 7 octets (4-byte header + 3-byte minimum TPDU).
  - `Decode` and `Parse` return payload slices that alias the input buffer; `Reader.ReadFrame` returns a newly allocated payload slice.
  - Comprehensive tests for happy paths, malformed frames, streaming truncation, size limits, aliasing, and writer short-write behavior.
  - Fuzz targets for `Decode` and `Parse`.

- **Tooling and docs**
  - `doc.go` with clear package documentation and scope.
  - `README.md` with badges, install instructions, examples, and size/ownership notes.
  - `LICENSE` (MIT).
  - `Makefile` with `test`, `vet`, `fuzz`, and `bench` targets.
  - GitHub Actions workflow (`.github/workflows/test.yml`) running tests, race tests, coverage, and Codecov upload on Go  and 1.23.

---