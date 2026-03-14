# go-tpkt Public API

Reference for the public API of `github.com/otfabric/go-tpkt`. See [README.md](README.md) for usage and [doc.go](doc.go) for package-level documentation.

---

## Constants

### Version

```go
const Version byte = 3
```

TPKT protocol version supported by this package (RFC 1006).

### HeaderLength

```go
const HeaderLength = 4
```

Length in octets of the fixed TPKT header.

### MinPacketLength

```go
const MinPacketLength = 7
```

Minimum legal TPKT total packet length (4-byte header + 3-byte minimum TPDU per RFC 1006).

### MaxPacketLength

```go
const MaxPacketLength = 65535
```

Maximum legal TPKT total packet length (16-bit length field).

---

## Types

### Frame

```go
type Frame struct {
    Payload []byte
}
```

Represents a single TPKT-framed TPDU. `Payload` is opaque to this package. A `Frame` value does not guarantee RFC 1006 validity; validity is enforced when marshaling.

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `Len` | `func (f Frame) Len() int` | Returns `HeaderLength + len(Payload)` (total wire length if encodable). |
| `MarshalBinary` | `func (f Frame) MarshalBinary() ([]byte, error)` | Encodes the frame into a full TPKT packet; returns error if size &lt; MinPacketLength or &gt; MaxPacketLength. |

---

### Reader

```go
type Reader struct {
    // r and maxFrameSize are unexported
}
```

Reads TPKT-framed payloads from an `io.Reader`. Not safe for concurrent use without external synchronization.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewReader` | `func NewReader(r io.Reader, opts ...ReaderOption) *Reader` | Builds a reader over `r`; default max packet size is `MaxPacketLength`. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `ReadFrame` | `func (r *Reader) ReadFrame() ([]byte, error)` | Reads the next TPKT frame and returns its payload (newly allocated). Returns `io.EOF` at end of stream, or an error wrapping `io.ErrUnexpectedEOF` on truncated data. |

---

### ReaderOption

```go
type ReaderOption func(*Reader)
```

Functional option for configuring a `Reader`.

| Function | Signature | Description |
|----------|-----------|-------------|
| `WithMaxFrameSize` | `func WithMaxFrameSize(n int) ReaderOption` | Sets max total TPKT size (header + payload). Values ≤ 0 keep default; values &lt; MinPacketLength are clamped to MinPacketLength. |

---

### Writer

```go
type Writer struct {
    // w is unexported
}
```

Writes payloads as TPKT-framed packets to an `io.Writer`. Not safe for concurrent use without external synchronization.

**Constructor:**

| Function | Signature | Description |
|----------|-----------|-------------|
| `NewWriter` | `func NewWriter(w io.Writer) *Writer` | Builds a writer over `w`. |

**Methods:**

| Method | Signature | Description |
|--------|-----------|-------------|
| `WriteFrame` | `func (w *Writer) WriteFrame(payload []byte) (int, error)` | Encodes `payload` as TPKT and writes the full packet. Returns total bytes written (header + payload), or an error (e.g. short write, Encode validation failure). |

---

## Functions

### Encode

```go
func Encode(payload []byte) ([]byte, error)
```

Builds a complete TPKT packet from `payload`. Returns a new slice (4-byte header + payload). Fails with `ErrInvalidLength` if total length &lt; MinPacketLength, or `ErrFrameTooLarge` if &gt; MaxPacketLength.

### Decode

```go
func Decode(pkt []byte) ([]byte, error)
```

Validates a complete TPKT packet and returns only the payload. The returned slice **aliases** `pkt`; copy if you need to retain or mutate independently.

### Parse

```go
func Parse(pkt []byte) (Frame, error)
```

Validates a complete TPKT packet and returns a `Frame`. `Frame.Payload` **aliases** `pkt`; copy if you need to retain or mutate independently.

---

## Sentinel errors

Use `errors.Is(err, target)` to classify. All are returned wrapped with context (e.g. `decode tpkt: tpkt header: version=2: tpkt: invalid version`).

| Variable | Description |
|----------|-------------|
| `ErrTooShort` | Buffer shorter than the 4-byte TPKT header. |
| `ErrInvalidVersion` | Header version byte is not 3. |
| `ErrInvalidReserved` | Header reserved byte is not 0. |
| `ErrInvalidLength` | Declared length is outside [MinPacketLength, MaxPacketLength] or otherwise invalid. |
| `ErrLengthMismatch` | Declared TPKT length does not match the actual buffer length. |
| `ErrFrameTooLarge` | Frame exceeds a configured or protocol maximum (e.g. Encode above MaxPacketLength, or Reader above WithMaxFrameSize). |
