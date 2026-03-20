# v0.1.1

**Changed**: Lowered minimum required Go version to 1.21 (was 1.22). All documentation, CI, and go.mod references updated accordingly. No code changes.

This release ensures compatibility with Go 1.21. No new features or bugfixes are included.
# go-tpkt Releases

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
  - GitHub Actions workflow (`.github/workflows/test.yml`) running tests, race tests, coverage, and Codecov upload on Go 1.21 and 1.23.

---