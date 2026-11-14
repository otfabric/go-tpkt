# Security

## Reporting a vulnerability

If you believe you have found a security vulnerability in `go-tpkt`, please report it responsibly.

- **Do not** open a public GitHub issue for security-sensitive bugs.
- Email **security@otfabric.io**, or open a **private security advisory** on GitHub (the [Security](https://github.com/otfabric/go-tpkt/security) tab → **Advisories** → **Report a vulnerability**).

Include a clear description, steps to reproduce, affected versions, and the impact. We will acknowledge your report and work with you on a fix and disclosure timeline.

## Supported versions

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Older releases | No |

## Threat model

`go-tpkt` frames packets on a stream that the caller supplies, and it treats the peer on that stream as untrusted. `ReadPacket` and `DecodePacket` check the version octet and the declared length of every header before the length is used, and return an error for a header they do not accept.

TPKT has no authentication, integrity protection or encryption, and this package adds none. Protecting the connection (TLS, network separation) is the caller's concern. So are the number of connections and the time a call may block: the package does not dial, listen or set deadlines. The payload is opaque to the package and goes to the next layer unchecked.

## Resource limits

| Resource | Bound | Where it is set |
|---|---|---|
| Memory for a packet that is being received | One buffer of the declared length minus the 4-octet header. It is allocated when the header has been accepted, before the payload arrives: at most `MaxPacketLength` minus 4 octets for each `ReadPacket` call, 65531 by default | `ReaderConfig.MaxPacketLength`, passed to `NewReader`: 7 to 65535, and 0 selects 65535 |
| A header that declares more than the limit | Refused with `ErrPacketTooLarge`. Nothing of the declared size is allocated and the payload is not read | The same limit |
| Memory for a packet that is being sent | One buffer of the payload plus 4 octets for each `EncodePacket` or `WritePacket` call, at most 65535 octets | The caller's payload; the protocol maximum is fixed |
| Memory for `DecodePacket` | None: the result aliases the input | — |
| Time a `ReadPacket` or `WritePacket` call blocks | Not bounded by the package | Deadlines on the connection, set by the caller |
| Connections, and packets per connection | Not bounded by the package | The caller |

A peer can make one `Reader` hold a payload buffer of up to the limit by sending a 4-octet header, for as long as the caller lets `ReadPacket` wait for the payload. A caller that does not need full-size packets lowers `MaxPacketLength`. A caller that accepts many connections also sets read deadlines and limits the connections. When a deadline expires inside a packet, the `Reader` refuses every later call with `ErrStreamMisaligned`, and the caller closes the connection ([API.md](API.md#state-of-the-stream-after-an-error)).

The limit and its allocation bound are tested by `TestReaderLimitBoundaries`, `TestReaderOversizeAllocationBudget` and `TestReaderAllocationBoundedByLimit` ([TESTING.md](TESTING.md#boundary-limit-and-stream-tests)); `FuzzReaderStream` gives arbitrary octets to `ReadPacket`.
