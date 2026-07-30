# go-tpkt Releases

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