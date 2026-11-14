# Architecture

go-tpkt is one package, `tpkt`, at the module root. It imports the standard library only (`encoding/binary`, `errors`, `fmt`, `io`). There is no `internal/` tree, no `cmd/` tree and no platform-specific code.

## Layers

```text
 caller (for example go-cotp), owns the connection
        |  opaque payload bytes
        v
 Reader.ReadPacket          Writer.WritePacket     reader.go, writer.go
        |                          |
        v                          v
 decodeHeader               EncodePacket           tpkt.go
 (also used by DecodePacket)
        |                          |
        v                          v
 io.Reader                  io.Writer              supplied by the caller
```

| File | Responsibility |
|---|---|
| `tpkt.go` | `EncodePacket`, `DecodePacket` and the header check `decodeHeader` |
| `reader.go` | `Reader`, `ReaderConfig`, EOF classification, the failed state after an error inside a packet |
| `writer.go` | `Writer`, retry of short writes, the failed state after an error inside a packet |
| `errors.go` | Protocol constants and exported sentinels ([ERRORS.md](ERRORS.md)) |

The buffer functions and the stream types share `decodeHeader` and `EncodePacket`, so both apply the same validation.

## Boundaries

The package treats a payload as opaque: it checks the version octet and the total length, and it does not read the TPDU. It does not dial, listen or set deadlines; the caller owns the `io.Reader` / `io.Writer` and closes it. Out-of-scope items are listed in [LIMITATIONS.md](LIMITATIONS.md).

## Goroutines and ownership

The package starts no goroutines and holds no global mutable state. All calls run on the caller's goroutine and block for as long as the underlying reader or writer blocks.

`Reader` and `Writer` keep no buffer between packets: `Reader` holds the stream and its size limit, `Writer` holds the stream. The only state each keeps is the error of a call that failed inside a packet; from then on it refuses every call with `ErrStreamMisaligned` ([API.md](API.md#state-of-the-stream-after-an-error)). One `Reader` and one `Writer` may run concurrently on the same full-duplex connection; one instance must not be shared between goroutines without external synchronization.

Slice ownership differs per entry point: `DecodePacket` returns a slice that aliases its input, while `EncodePacket` and `ReadPacket` return newly allocated slices ([API.md](API.md#ownership)).
