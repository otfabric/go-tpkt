# Errors

The errors of this package are sentinels. Use `errors.Is`; none is retryable as is, because each reports invalid input or a caller mistake. `ReadPacket` and `WritePacket` also return errors that are not sentinels of this package: [Stream errors](#stream-errors).

| Error | Returned when |
|---|---|
| `ErrNilReader` | `NewReader` was called with a nil `io.Reader`, or `ReadPacket` was called on a zero-value or nil `Reader` |
| `ErrNilWriter` | `NewWriter` was called with a nil `io.Writer`, or `WritePacket` was called on a zero-value or nil `Writer` |
| `ErrTooShort` | A buffer is shorter than the 4-byte TPKT header |
| `ErrInvalidVersion` | The version byte is not 3 |
| `ErrInvalidPacketLength` | The length a header declares is outside `[MinPacketLength, MaxPacketLength]` |
| `ErrInvalidMaxPacketLength` | `ReaderConfig.MaxPacketLength` is outside that range (zero means the default) |
| `ErrLengthMismatch` | The declared length does not match the size of the input buffer |
| `ErrPacketTooLarge` | A packet exceeds the reader's configured maximum |
| `ErrPayloadTooShort` | An encode payload is shorter than `MinPayloadLength` (3 octets) |
| `ErrPayloadTooLarge` | An encode payload exceeds `MaxPayloadLength` (65531 octets) |
| `ErrStreamMisaligned` | An earlier `ReadPacket` or `WritePacket` on the same `Reader` or `Writer` failed after one or more octets of a packet had moved. The stream is no longer aligned on a packet boundary, and the `Reader` or `Writer` returns this error from every later call, without touching the stream. The error also wraps the error of the earlier call, so `errors.Is` finds both. Close the connection |

## Stream errors

`ReadPacket` and `WritePacket` return these errors as well. They come from the standard library or from the stream the caller supplied.

| Error | Returned when | Retryable |
|---|---|---|
| `io.EOF` | `ReadPacket` found the end of the stream on a packet boundary. It is returned as is, not wrapped | No: the stream has ended. The `Reader` stays usable and returns `io.EOF` again |
| `io.ErrUnexpectedEOF` | The stream ended inside a packet: after one to three octets of a header, or before the whole payload. Also when the stream wraps its EOF in an error of its own | No: the `Reader` stays failed. Close the connection |
| `io.ErrShortWrite` | The underlying writer accepted no octet of a `Write` and returned no error | As for an error of the stream |
| An error of the stream (a deadline, a reset, a closed connection) | The underlying `io.Reader` or `io.Writer` returned it. It is wrapped with `%w`, so `errors.Is` and `errors.As` still find it: a read deadline matches `os.ErrDeadlineExceeded` | When no octet of the packet had moved, the call can be repeated. Otherwise the `Reader` or `Writer` stays failed and the next call returns `ErrStreamMisaligned` ([API.md](API.md#state-of-the-stream-after-an-error)) |

An encode error (`ErrPayloadTooShort`, `ErrPayloadTooLarge`) from `WritePacket` is returned before anything is written, in every state of the `Writer`.
