# go-tpkt API Reference

Complete reference for the public API of [`github.com/otfabric/go-tpkt`](https://pkg.go.dev/github.com/otfabric/go-tpkt).

For installation and a quick getting started guide, see [README.md](README.md). For package-level design notes, see [doc.go](doc.go).

---

## Table of contents

- [Overview](#overview)
- [Wire format](#wire-format)
- [Constants](#constants)
- [Types](#types)
  - [Frame](#frame)
  - [Reader](#reader)
  - [ReaderOption](#readeroption)
  - [Writer](#writer)
- [Package functions](#package-functions)
  - [Encode](#encode)
  - [Decode](#decode)
  - [Parse](#parse)
- [Errors](#errors)
- [Validation and size limits](#validation-and-size-limits)
- [Payload ownership](#payload-ownership)
- [Streaming semantics](#streaming-semantics)
- [Concurrency](#concurrency)
- [Usage patterns](#usage-patterns)
- [Quick reference](#quick-reference)

---

## Overview

`tpkt` implements **TPKT framing** as defined in [RFC 1006](https://datatracker.ietf.org/doc/html/rfc1006) section 6. TPKT wraps an opaque payload (typically a TPDU from a higher-level ISO transport protocol) in a fixed 4-byte header so it can be carried over a TCP byte stream.

This package is intentionally limited to framing. It does **not** parse or construct:

- COTP or CR/CC/DT TPDU semantics
- TSAP addressing or ISO session/presentation layers
- S7comm, MMS, IEC 61850, or other application protocols

Callers build higher-level stacks on top of the payload bytes returned by `Decode`, `Parse`, and `Reader.ReadFrame`, and send payload bytes through `Writer.WriteFrame`.

### Public API surface

| Category | Symbols |
|----------|---------|
| Constants | `Version`, `HeaderLength`, `MinPacketLength`, `MaxPacketLength` |
| Types | `Frame`, `Reader`, `Writer`, `ReaderOption` |
| Constructors | `NewReader`, `NewWriter` |
| Options | `WithMaxFrameSize` |
| Codec helpers | `Encode`, `Decode`, `Parse` |
| Methods | `Frame.Len`, `Frame.MarshalBinary`, `Reader.ReadFrame`, `Writer.WriteFrame` |
| Errors | `ErrTooShort`, `ErrInvalidVersion`, `ErrInvalidReserved`, `ErrInvalidLength`, `ErrLengthMismatch`, `ErrFrameTooLarge` |

---

## Wire format

Every TPKT packet on the wire has this layout:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    version    |   reserved    |          length (BE)          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          payload (TPDU)                       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

| Field | Size | Value | Notes |
|-------|------|-------|-------|
| `version` | 1 byte | `3` | Must equal `tpkt.Version`. |
| `reserved` | 1 byte | `0` | Must be zero. |
| `length` | 2 bytes | `7`–`65535` | Big-endian total packet size in octets, **including** the 4-byte header. |
| `payload` | `length − 4` | opaque | Interpreted by higher-level protocols. |

A minimum packet length of **7** octets means the smallest legal payload is **3** bytes (4-byte header + 3-byte minimum TPDU for transport class 0 per RFC 1006).

---

## Constants

### `Version`

```go
const Version byte = 3
```

The TPKT protocol version supported by this package. RFC 1006 defines version 3 for this framing scheme. `Encode` always writes this value; `Decode`, `Parse`, and `Reader.ReadFrame` reject packets with any other version byte.

### `HeaderLength`

```go
const HeaderLength = 4
```

The fixed size of the TPKT header in octets. Use this when sizing buffers or slicing header fields from a raw packet.

### `MinPacketLength`

```go
const MinPacketLength = 7
```

The minimum legal **total** TPKT packet length on the wire (header + payload). Equivalently, the minimum encodable payload length is `MinPacketLength - HeaderLength` = **3** bytes.

Packets declaring a length below 7 — including a 4-byte header with no TPDU bytes — are rejected.

### `MaxPacketLength`

```go
const MaxPacketLength = 65535
```

The maximum legal **total** TPKT packet length, determined by the 16-bit big-endian length field. The maximum encodable payload size is `MaxPacketLength - HeaderLength` = **65531** bytes.

---

## Types

### `Frame`

```go
type Frame struct {
    Payload []byte
}
```

Represents a single TPKT-framed TPDU. `Payload` is treated as an opaque byte sequence; this package does not interpret its contents.

A `Frame` value by itself does not guarantee RFC 1006 validity. Payloads that are too short or too large are rejected when the frame is encoded via `MarshalBinary` or `Encode`.

#### `Frame.Len`

```go
func (f Frame) Len() int
```

Returns `HeaderLength + len(f.Payload)`, i.e. the total on-the-wire length the frame would occupy **if** it is encodable. This method does not validate the payload; it only reports the computed size.

#### `Frame.MarshalBinary`

```go
func (f Frame) MarshalBinary() ([]byte, error)
```

Encodes the frame into a complete TPKT packet. Equivalent to `Encode(f.Payload)`.

- Returns a **newly allocated** slice containing the 4-byte header followed by the payload.
- Returns an error if the encoded packet would be smaller than `MinPacketLength` (`ErrInvalidLength`) or larger than `MaxPacketLength` (`ErrFrameTooLarge`).

**Example:**

```go
frame := tpkt.Frame{Payload: []byte{0x02, 0xf0, 0x80}}
pkt, err := frame.MarshalBinary()
if err != nil {
    log.Fatal(err)
}
// pkt is a complete TPKT packet ready to send on the wire.
```

---

### `Reader`

```go
type Reader struct { /* unexported fields */ }
```

Reads TPKT-framed payloads sequentially from an underlying [`io.Reader`](https://pkg.go.dev/io#Reader). Suitable for streaming sources such as [`net.Conn`](https://pkg.go.dev/net#Conn), files, or in-memory buffers.

`Reader` is **not** safe for concurrent use from multiple goroutines without external synchronization.

#### `NewReader`

```go
func NewReader(r io.Reader, opts ...ReaderOption) *Reader
```

Constructs a `Reader` over `r`.

- By default, accepts packets up to `MaxPacketLength` (65535 octets total).
- Optional `ReaderOption` values (such as `WithMaxFrameSize`) can impose a stricter upper bound.

**Example:**

```go
r := tpkt.NewReader(conn)
// or with a size limit:
r := tpkt.NewReader(conn, tpkt.WithMaxFrameSize(8192))
```

#### `Reader.ReadFrame`

```go
func (r *Reader) ReadFrame() ([]byte, error)
```

Reads the next complete TPKT frame from the stream and returns its **payload** (header stripped).

**Behavior:**

1. Reads exactly 4 header bytes via `io.ReadFull`.
2. Validates version, reserved byte, and declared length (same rules as `Decode`).
3. Rejects frames whose declared total length exceeds the reader's `maxFrameSize` (`ErrFrameTooLarge`).
4. Allocates and reads the payload into a **new slice**.

**Return values:**

| Result | Meaning |
|--------|---------|
| `(payload, nil)` | One complete frame was read. |
| `(nil, io.EOF)` | Called at a frame boundary and the underlying reader has no more data (clean end of stream). |
| `(nil, err)` | Malformed header, truncated stream, oversized frame, or other I/O failure. Truncated reads wrap [`io.ErrUnexpectedEOF`](https://pkg.go.dev/io#ErrUnexpectedEOF). |

**Example — single frame:**

```go
payload, err := r.ReadFrame()
if err != nil {
    if errors.Is(err, io.EOF) {
        return // peer closed cleanly between frames
    }
    return fmt.Errorf("read frame: %w", err)
}
// payload is a newly allocated TPDU byte slice.
```

**Example — read loop:**

```go
for {
    payload, err := r.ReadFrame()
    if err != nil {
        if errors.Is(err, io.EOF) {
            break
        }
        return err
    }
    if err := handleTPDU(payload); err != nil {
        return err
    }
}
```

The reader works correctly even when the underlying `io.Reader` returns data one byte at a time.

---

### `ReaderOption`

```go
type ReaderOption func(*Reader)
```

Functional option type for configuring a `Reader`. Pass options to `NewReader`.

#### `WithMaxFrameSize`

```go
func WithMaxFrameSize(n int) ReaderOption
```

Sets an upper bound on the **total** TPKT packet size (header + payload) that `ReadFrame` will accept.

| Input | Effect |
|-------|--------|
| `n <= 0` | No change; default `MaxPacketLength` remains in effect. |
| `0 < n < MinPacketLength` | Clamped up to `MinPacketLength` (7). |
| `n >= MinPacketLength` | `ReadFrame` rejects frames with declared total length **greater than** `n`. |

`WithMaxFrameSize` is the **only** caller-configurable size limit in this package. `Encode`, `Decode`, and `Parse` enforce protocol bounds only.

**Example — protect against oversized frames before allocation:**

```go
const maxTPKT = 4096
r := tpkt.NewReader(conn, tpkt.WithMaxFrameSize(maxTPKT))
```

---

### `Writer`

```go
type Writer struct { /* unexported fields */ }
```

Writes payloads as TPKT-framed packets to an underlying [`io.Writer`](https://pkg.go.dev/io#Writer). To send a `Frame`, pass `f.Payload` explicitly to `WriteFrame`.

`Writer` is **not** safe for concurrent use from multiple goroutines without external synchronization.

#### `NewWriter`

```go
func NewWriter(w io.Writer) *Writer
```

Constructs a `Writer` over `w`.

#### `Writer.WriteFrame`

```go
func (w *Writer) WriteFrame(payload []byte) (int, error)
```

Encodes `payload` as a TPKT packet and writes the **entire** packet to the underlying writer.

**Returns:**

- `int` — total octets written (header + payload) on success, or the number written before an error on partial failure.
- `error` — encoding failure, underlying write error, or [`io.ErrShortWrite`](https://pkg.go.dev/io#ErrShortWrite) if the writer returns zero bytes without an error.

The writer loops until all bytes are written or an error occurs, so callers do not need to handle partial TPKT packets themselves.

**Example:**

```go
w := tpkt.NewWriter(conn)
n, err := w.WriteFrame(tpdu)
if err != nil {
    return fmt.Errorf("write frame: %w", err)
}
// n == len(tpdu) + tpkt.HeaderLength
```

---

## Package functions

### `Encode`

```go
func Encode(payload []byte) ([]byte, error)
```

Builds a complete TPKT packet from `payload`.

**Returns:**

- A **newly allocated** slice: 4-byte header + payload.
- Header fields: `version=3`, `reserved=0`, `length=HeaderLength+len(payload)` (big-endian).

**Errors:**

| Condition | Sentinel |
|-----------|----------|
| `len(payload) < 3` (total length &lt; 7) | `ErrInvalidLength` |
| Total length &gt; 65535 | `ErrFrameTooLarge` |

**Example:**

```go
pkt, err := tpkt.Encode([]byte{0x02, 0xf0, 0x80})
// pkt == []byte{0x03, 0x00, 0x00, 0x07, 0x02, 0xf0, 0x80}
```

---

### `Decode`

```go
func Decode(pkt []byte) ([]byte, error)
```

Validates a complete TPKT packet in `pkt` and returns **only the payload**.

**Validation performed:**

1. Buffer is at least `HeaderLength` (4) bytes.
2. Version byte is `3`.
3. Reserved byte is `0`.
4. Declared length is in `[MinPacketLength, MaxPacketLength]`.
5. Declared length equals `len(pkt)`.

**Ownership:** the returned slice **aliases** `pkt`. Copy it if you need to retain or mutate the payload independently of the input buffer.

**Example:**

```go
pkt := []byte{0x03, 0x00, 0x00, 0x07, 'a', 'b', 'c'}
payload, err := tpkt.Decode(pkt)
// payload == []byte{'a', 'b', 'c'}
// payload aliases pkt[4:7]
```

---

### `Parse`

```go
func Parse(pkt []byte) (Frame, error)
```

Validates a complete TPKT packet and returns a `Frame` with the payload populated. Performs the same validation as `Decode`.

**Ownership:** `Frame.Payload` **aliases** `pkt`. Copy it if you need an independent slice.

**When to use `Parse` vs `Decode`:**

- Use `Decode` when you only need the payload bytes.
- Use `Parse` when you want a `Frame` value (e.g. to call `Len()` or `MarshalBinary()` without re-parsing).

**Example:**

```go
frame, err := tpkt.Parse(pkt)
if err != nil {
    return err
}
fmt.Println(frame.Len()) // total wire length if re-encoded
```

---

## Errors

All exported errors are sentinel values. Classify failures with [`errors.Is`](https://pkg.go.dev/errors#Is):

```go
payload, err := tpkt.Decode(pkt)
if err != nil {
    switch {
    case errors.Is(err, tpkt.ErrInvalidVersion):
        // peer sent an unsupported TPKT version
    case errors.Is(err, tpkt.ErrLengthMismatch):
        // declared length does not match buffer size
    default:
        return fmt.Errorf("decode: %w", err)
    }
}
```

Errors are typically wrapped with context (e.g. `decode tpkt: tpkt header: version=2: tpkt: invalid version`). Always use `errors.Is`, not string comparison.

### Sentinel errors

| Variable | When returned |
|----------|---------------|
| `ErrTooShort` | Buffer is shorter than the 4-byte TPKT header. |
| `ErrInvalidVersion` | Header version byte is not `3`. |
| `ErrInvalidReserved` | Header reserved byte is not `0`. |
| `ErrInvalidLength` | Declared length is below `MinPacketLength`, above `MaxPacketLength`, or the payload is too short to encode. |
| `ErrLengthMismatch` | Declared TPKT length does not equal the actual buffer length. |
| `ErrFrameTooLarge` | Frame exceeds `MaxPacketLength` on encode, or exceeds `Reader`'s `WithMaxFrameSize` limit on read. |

### I/O errors from `Reader` and `Writer`

| Error | Source |
|-------|--------|
| `io.EOF` | `ReadFrame` at a clean frame boundary with no remaining data. |
| `io.ErrUnexpectedEOF` | Stream ends mid-header or mid-payload (wrapped in a descriptive error). |
| `io.ErrShortWrite` | Underlying writer returned 0 bytes without an error during `WriteFrame`. |
| Underlying write/read errors | Propagated from the `io.Reader` / `io.Writer` (wrapped). |

---

## Validation and size limits

### Encode-time limits

| Check | Bound |
|-------|-------|
| Minimum total packet size | `MinPacketLength` (7) → minimum payload 3 bytes |
| Maximum total packet size | `MaxPacketLength` (65535) → maximum payload 65531 bytes |

Empty payloads and 1–2 byte payloads are rejected at encode time.

### Decode / Parse limits

`Decode` and `Parse` validate protocol structure only:

- Version, reserved, and length fields
- Consistency between declared length and buffer size

They do **not** accept a caller-configurable maximum frame size. To cap incoming frame size before payload allocation, use `Reader` with `WithMaxFrameSize`.

### Reader limits

`Reader` applies the same structural validation as `Decode`, plus:

- Rejects frames whose declared total length exceeds `maxFrameSize` (default `MaxPacketLength`, configurable via `WithMaxFrameSize`).

---

## Payload ownership

| API | Returned payload | Safe to mutate? | Aliases input? |
|-----|------------------|-----------------|----------------|
| `Decode(pkt)` | `[]byte` | Only if you copy first | Yes — aliases `pkt` |
| `Parse(pkt)` | `Frame.Payload` | Only if you copy first | Yes — aliases `pkt` |
| `Reader.ReadFrame()` | `[]byte` | Yes — newly allocated | No |
| `Encode(payload)` | full packet `[]byte` | Yes — newly allocated | No |

**Copy when retaining decoded data:**

```go
payload, err := tpkt.Decode(pkt)
if err != nil {
    return err
}
owned := append([]byte(nil), payload...) // independent copy
```

---

## Streaming semantics

### Multiple frames on one stream

TPKT over TCP is a **sequence of length-prefixed frames**. A single TCP connection typically carries many back-to-back TPKT packets. `Reader.ReadFrame` reads one frame at a time; call it in a loop until `io.EOF`.

### End-of-stream behavior

| Situation | `ReadFrame` result |
|-----------|-------------------|
| Empty stream (no bytes) | `io.EOF` |
| Complete frame(s), then no more data | Last frame returns `(payload, nil)`; next call returns `io.EOF` |
| Truncated header (1–3 bytes then EOF) | Error wrapping `io.ErrUnexpectedEOF` |
| Full header, truncated payload | Error wrapping `io.ErrUnexpectedEOF` |

Distinguish clean shutdown (`io.EOF` between frames) from protocol errors (malformed or truncated data) using `errors.Is`.

### Partial underlying I/O

`Reader` uses `io.ReadFull` for both header and payload, so it handles `io.Reader` implementations that return data in arbitrary chunk sizes (including one byte at a time).

`Writer` loops on `Write` until the full packet is sent or an error occurs, handling short writes from the underlying writer.

---

## Concurrency

`Reader` and `Writer` are **not** safe for concurrent use from multiple goroutines. If multiple goroutines share a connection, protect the reader/writer with a mutex or confine I/O to a single goroutine.

Package-level functions (`Encode`, `Decode`, `Parse`) and `Frame` methods are safe for concurrent use because they do not hold mutable state.

---

## Usage patterns

### Buffer-oriented encode/decode

Use when you already have a complete packet in memory (e.g. from a capture, test fixture, or datagram):

```go
pkt, err := tpkt.Encode(tpdu)
decoded, err := tpkt.Decode(pkt)
```

### Stream-oriented read/write

Use over a live TCP connection or any streaming `io` source:

```go
r := tpkt.NewReader(conn, tpkt.WithMaxFrameSize(8192))
w := tpkt.NewWriter(conn)

for {
    tpdu, err := r.ReadFrame()
    if err != nil {
        if errors.Is(err, io.EOF) {
            break
        }
        return err
    }
    if _, err := w.WriteFrame(responseFor(tpdu)); err != nil {
        return err
    }
}
```

### In-memory testing

Use `bytes.Buffer` and `bytes.NewReader` without a network:

```go
var buf bytes.Buffer
w := tpkt.NewWriter(&buf)
_, _ = w.WriteFrame([]byte{0x01, 0x02, 0x03})

r := tpkt.NewReader(bytes.NewReader(buf.Bytes()))
payload, _ := r.ReadFrame()
```

### Building on higher-level protocols

This package stops at TPKT framing. A typical stack looks like:

```
TCP stream → tpkt.Reader → TPDU bytes → COTP parser → application protocol
Application  → COTP builder → TPDU bytes → tpkt.Writer → TCP stream
```

Payload bytes are opaque to `go-tpkt`; your COTP, S7comm, or MMS layer owns their interpretation.

---

## Quick reference

| Symbol | Kind | Summary |
|--------|------|---------|
| `Version` | const | TPKT version byte (`3`) |
| `HeaderLength` | const | Header size in octets (`4`) |
| `MinPacketLength` | const | Minimum total packet size (`7`) |
| `MaxPacketLength` | const | Maximum total packet size (`65535`) |
| `Frame` | type | TPKT-framed TPDU container |
| `Frame.Len` | method | Wire length if encodable |
| `Frame.MarshalBinary` | method | Encode frame to packet bytes |
| `Reader` | type | Streaming TPKT frame reader |
| `NewReader` | func | Construct a `Reader` |
| `Reader.ReadFrame` | method | Read next payload from stream |
| `ReaderOption` | type | Functional option for `Reader` |
| `WithMaxFrameSize` | func | Cap accepted frame size |
| `Writer` | type | Streaming TPKT frame writer |
| `NewWriter` | func | Construct a `Writer` |
| `Writer.WriteFrame` | method | Encode and write one frame |
| `Encode` | func | Build packet from payload |
| `Decode` | func | Validate packet, return payload |
| `Parse` | func | Validate packet, return `Frame` |
| `ErrTooShort` | var | Buffer shorter than header |
| `ErrInvalidVersion` | var | Version byte ≠ 3 |
| `ErrInvalidReserved` | var | Reserved byte ≠ 0 |
| `ErrInvalidLength` | var | Length field out of range |
| `ErrLengthMismatch` | var | Declared length ≠ buffer size |
| `ErrFrameTooLarge` | var | Frame exceeds size limit |
