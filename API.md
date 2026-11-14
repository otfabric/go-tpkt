# go-tpkt API Reference (v1)

Complete reference for the public API of [`github.com/otfabric/go-tpkt`](https://pkg.go.dev/github.com/otfabric/go-tpkt).

For installation and a quick getting started guide, see [README.md](README.md).

---

## Table of contents

- [Overview](#overview)
- [TPKT requirements matrix](#tpkt-requirements-matrix)
- [Wire format](#wire-format)
- [Constants](#constants)
- [Byte-buffer API](#byte-buffer-api)
- [Ownership](#ownership)
- [Stream API](#stream-api)
- [Errors](#errors)
- [EOF and stream contracts](#eof-and-stream-contracts)
- [Concurrency](#concurrency)
- [Usage patterns](#usage-patterns)

---

## Overview

`tpkt` provides **TPKT framing for RFC 1006 and RFC 2126 transport profiles**.

Claims (exact wording):

- TPKT framing compliant with RFC 1006 §6 and RFC 2126 §4.3/§6.10.
- Does **not** claim full RFC 2126 / ITOT compliance (Class 0/2 COTP, dual TCP, addressing, etc. are out of scope).

The library enforces the RFC 1006 minimum total packet length of 7 octets but treats the TPDU as **opaque**. COTP validates TPDU structure.

| Category | Symbols |
|----------|---------|
| Constants | `Version`, `HeaderLength`, `MinPacketLength`, `MaxPacketLength`, `MinPayloadLength`, `MaxPayloadLength` |
| Buffer API | `EncodePacket`, `DecodePacket` |
| Stream API | `Reader`, `ReaderConfig`, `NewReader`, `ReadPacket`, `Writer`, `NewWriter`, `WritePacket` |
| Errors | see [Errors](#errors) |

---

## TPKT requirements matrix

| ID | Requirement or contract | Basis | Status |
|----|-------------------------|-------|--------|
| TPKT-1006-01 | Version field is 3 | RFC normative (§6) | Pass |
| TPKT-1006-02 | Total length is 16-bit big-endian, includes header | RFC normative (§6) | Pass |
| TPKT-1006-03 | Total length is 7–65535 | RFC normative (§6) | Pass |
| TPKT-1006-D01 | Maximum TPDU is 65531 | Derived | Pass |
| TPKT-1006-D02 | Min length implies ≥3-byte opaque payload; TPDU not inspected | Derived + layer boundary | Pass |
| TPKT-2126-01 | Output reserved value is 0 | RFC normative (§4.3) | Pass |
| TPKT-2126-02 | Reserved input is ignored | RFC recommendation (§6.10 SHOULD) | Pass |
| TPKT-SAFE-01 | Length validated before allocation | Security invariant | Pass |
| TPKT-API-01 | DecodePacket rejects trailing bytes | API contract | Pass |
| TPKT-API-02 | EOF after accepted header → UnexpectedEOF | API contract | Pass |
| TPKT-API-03 | Stream unusable after reject (no drain) | API contract | Pass |
| TPKT-API-04 | Reader∥Writer OK on full-duplex; instances not shared | API contract | Pass |
| TPKT-API-05 | Invalid MaxPacketLength rejected (never clamped) | API contract | Pass |
| TPKT-API-06 | After an error inside a packet, `Reader` and `Writer` refuse every later call with `ErrStreamMisaligned`; an error on a packet boundary leaves them usable | API contract | Pass |

---

## Wire format

```
| version (8) | reserved (8) | packet length (16 BE) | TPDU (variable) |
```

| Field | Value |
|-------|-------|
| version | 3 |
| reserved | written 0; ignored on input |
| packet length | total octets including header, 7…65535 |

---

## Constants

```go
const (
    Version          byte = 3
    HeaderLength          = 4
    MinPacketLength       = 7
    MaxPacketLength       = 65535
    MinPayloadLength      = 3
    MaxPayloadLength      = 65531
)
```

---

## Byte-buffer API

### `EncodePacket`

```go
func EncodePacket(payload []byte) ([]byte, error)
```

Builds exactly one TPKT. Reserved is always 0. Payload must be in `[MinPayloadLength, MaxPayloadLength]`.

### `DecodePacket`

```go
func DecodePacket(packet []byte) ([]byte, error)
```

Validates exactly one complete TPKT and returns the payload (aliases `packet`; its capacity ends at the end of the packet). Trailing bytes → `ErrLengthMismatch`. Reserved ignored on input.

---

## Ownership

| API | Buffer ownership |
|-----|------------------|
| `EncodePacket` | Returns a **new** packet buffer. The input `payload` is copied; the caller retains ownership of `payload`. |
| `DecodePacket` | Returned payload **aliases** `packet` (a subslice). Mutating or reusing `packet` after decode affects the result; copy the payload if it must outlive the input buffer. The capacity of the payload ends at the end of the packet, so `append` onto it copies and does not write into the buffer behind `packet`. |
| `ReadPacket` | Returns a **newly allocated** payload. It does not alias an internal stream buffer. |
| `WritePacket` | Copies `payload` into the encoded TPKT written to the underlying `io.Writer`. The caller retains ownership of `payload`. |

---

## Stream API

### `ReaderConfig`

```go
type ReaderConfig struct {
    MaxPacketLength int // 0 = default MaxPacketLength; else [7, 65535]
}
```

Invalid values return `ErrInvalidMaxPacketLength` from `NewReader` — never clamped.

### `NewReader` / `ReadPacket`

```go
func NewReader(r io.Reader, cfg ReaderConfig) (*Reader, error)
func (r *Reader) ReadPacket() ([]byte, error)
```

- Nil `r` → `ErrNilReader`
- Empty config → max = 65535
- Returns newly allocated payload
- Oversized declared length → `ErrPacketTooLarge`; stream unusable
- An error after one or more octets of a packet were consumed leaves the `Reader` failed: every later `ReadPacket` → `ErrStreamMisaligned` ([State of the stream after an error](#state-of-the-stream-after-an-error))
- An error on a packet boundary (`io.EOF`, a deadline on an idle connection) leaves the `Reader` usable
- The zero value of `Reader` is not usable: `ReadPacket` on a zero-value or nil `Reader` → `ErrNilReader`

### `NewWriter` / `WritePacket`

```go
func NewWriter(w io.Writer) (*Writer, error)
func (w *Writer) WritePacket(payload []byte) error
```

- Nil `w` → `ErrNilWriter`
- Short writes retried; `(0, nil)` → `io.ErrShortWrite`
- A payload outside `[MinPayloadLength, MaxPayloadLength]` is refused before anything is written, in every state of the `Writer`
- An error after one or more octets of a packet were written leaves the `Writer` failed: every later `WritePacket` → `ErrStreamMisaligned` ([State of the stream after an error](#state-of-the-stream-after-an-error))
- An error before any octet of a packet was written leaves the `Writer` usable
- The zero value of `Writer` is not usable: `WritePacket` on a zero-value or nil `Writer` → `ErrNilWriter`

---

## Errors

| Sentinel | Meaning |
|----------|---------|
| `ErrNilReader` / `ErrNilWriter` | Nil dependency at construction; `ReadPacket` / `WritePacket` on a zero-value or nil `Reader` / `Writer` |
| `ErrTooShort` | Buffer shorter than header |
| `ErrInvalidVersion` | Version ≠ 3 |
| `ErrInvalidPacketLength` | Declared wire length out of [7, 65535] |
| `ErrInvalidMaxPacketLength` | ReaderConfig max invalid |
| `ErrLengthMismatch` | Declared length ≠ buffer length (incl. trailing bytes) |
| `ErrPacketTooLarge` | Declared length > configured max |
| `ErrPayloadTooShort` / `ErrPayloadTooLarge` | Encode payload size |
| `ErrStreamMisaligned` | An earlier `ReadPacket` / `WritePacket` on the same `Reader` / `Writer` failed inside a packet; the error also wraps that first error |

Classify with `errors.Is`. `ReadPacket` and `WritePacket` also return `io.EOF`, `io.ErrUnexpectedEOF`, `io.ErrShortWrite` and the errors of the underlying stream, wrapped ([ERRORS.md](ERRORS.md#stream-errors)).

---

## EOF and stream contracts

| Condition | Result |
|-----------|--------|
| No header byte at packet boundary | `io.EOF` |
| Partial header then close | `io.ErrUnexpectedEOF` |
| Complete valid header, then close before full payload | `io.ErrUnexpectedEOF` |
| Partial payload then close | `io.ErrUnexpectedEOF` |

An end of the stream inside a packet is `io.ErrUnexpectedEOF` and never matches `io.EOF`, also when the stream wraps its EOF in an error of its own.

### State of the stream after an error

`Reader` and `Writer` record an error that leaves the stream inside a packet, and refuse every later call. An error on a packet boundary is not recorded.

| Error | Octets of a packet that moved | State afterwards | Next call on the same `Reader` or `Writer` |
|-------|-------------------------------|------------------|--------------------------------------------|
| `io.EOF` from `ReadPacket` | None | The stream ended on a packet boundary; the `Reader` stays usable | `io.EOF` again, or a packet if the stream delivers one |
| An error of the stream (a deadline, a reset) or `io.ErrShortWrite`, before any octet of the packet moved | None | Still on a packet boundary; the `Reader` or `Writer` stays usable | Proceeds as usual: a retry after a deadline on an idle connection reads or writes the next packet |
| `ErrPayloadTooShort`, `ErrPayloadTooLarge` from `WritePacket` | None | Unchanged: the payload is checked before the `Writer` looks at its stream | Proceeds as usual |
| `ErrInvalidVersion`, `ErrInvalidPacketLength`, `ErrPacketTooLarge` from `ReadPacket` | The header | Not aligned: the rest of the refused packet is not drained. The `Reader` is failed | `ErrStreamMisaligned` |
| `io.ErrUnexpectedEOF` from `ReadPacket` | Part of a packet | The stream ended inside a packet. The `Reader` is failed | `ErrStreamMisaligned` |
| An error of the stream, or `io.ErrShortWrite`, after one or more octets of the packet moved | One or more | Not aligned: the octets that `ReadPacket` consumed are discarded; after `WritePacket` an incomplete packet is on the stream. The `Reader` or `Writer` is failed | `ErrStreamMisaligned` |

The call that fails returns its error as it always did: the sentinel, or the error of the stream wrapped with `%w`. A failed `Reader` or `Writer` then returns, from every later `ReadPacket` or `WritePacket`, an error that matches `ErrStreamMisaligned` and wraps the first error, so `errors.Is` and `errors.As` still find the cause (`os.ErrDeadlineExceeded`, `ErrInvalidVersion`). Only a payload-size error from `WritePacket` comes before it. A failed `Reader` or `Writer` does not read from or write to the stream again, and nothing resets it: the caller closes the connection.

A caller that sets deadlines may call `ReadPacket` or `WritePacket` again after a deadline error. On a packet boundary the call proceeds; inside a packet it returns `ErrStreamMisaligned`.

The failed state belongs to one `Reader` or one `Writer`. A `Reader` that failed does not stop the `Writer` on the same connection, and a second `Reader` built on the same stream would start inside a packet.

---

## Concurrency

A `Reader` and a `Writer` may be used concurrently on the same full-duplex connection. Individual `Reader` / `Writer` values are not safe for concurrent method calls.

---

## Usage patterns

```go
r, err := tpkt.NewReader(conn, tpkt.ReaderConfig{MaxPacketLength: 8192})
w, err := tpkt.NewWriter(conn)

for {
    payload, err := r.ReadPacket()
    if err != nil {
        if errors.Is(err, io.EOF) {
            break
        }
        return err
    }
    if err := w.WritePacket(responseFor(payload)); err != nil {
        return err
    }
}
```

On any error other than `io.EOF` the loop returns and the caller closes `conn`. A loop that went on instead would get `ErrStreamMisaligned` from the next call ([State of the stream after an error](#state-of-the-stream-after-an-error)).

Stack: `TCP → tpkt.Reader → opaque TPDU → COTP → application`.
