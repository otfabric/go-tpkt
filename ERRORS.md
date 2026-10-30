# Errors

All errors are sentinels. Use `errors.Is`; none is retryable as is, because each reports invalid input or a caller mistake.

| Error | Returned when |
|---|---|
| `ErrNilReader` | `NewReader` was called with a nil `io.Reader` |
| `ErrNilWriter` | `NewWriter` was called with a nil `io.Writer` |
| `ErrTooShort` | A buffer is shorter than the 4-byte TPKT header |
| `ErrInvalidVersion` | The version byte is not 3 |
| `ErrInvalidPacketLength` | A declared or computed length is outside `[MinPacketLength, MaxPacketLength]` |
| `ErrInvalidMaxPacketLength` | `ReaderConfig.MaxPacketLength` is outside that range (zero means the default) |
| `ErrLengthMismatch` | The declared length does not match the size of the input buffer |
| `ErrPacketTooLarge` | A packet exceeds the reader's configured maximum |
| `ErrPayloadTooShort` | An encode payload is shorter than `MinPayloadLength` (3 octets) |
| `ErrPayloadTooLarge` | An encode payload exceeds `MaxPayloadLength` (65531 octets) |
