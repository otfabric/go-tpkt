# Limitations

## Certification and verification

Nothing is certified. go-tpkt has no independent-implementation interoperability suite: its tests exchange packets only between its own `Writer` and `Reader`, or use hand-built byte sequences. No test runs against another TPKT stack or a device. The requirement results in [API.md](API.md) are results of this module's own tests, not of an external conformance test.

## Scope not implemented

- TPDU parsing: CR, CC, DT and other COTP TPDUs are opaque payloads.
- The ITOT / Class 0/2 transport profile of RFC 2126, including dual TCP expedited-data connections.
- TSAP / NSAPA addressing and ISO session or presentation layers.
- Application protocols (S7comm, MMS, IEC 61850).
- TCP dial, listen and port binding.

## Behaviour to account for

- **Timeouts:** the package has none. `ReadPacket` and `WritePacket` block until the underlying stream returns; set deadlines on the connection. Reconnection and retry after an error are also the caller's concern.
- **Read errors end the stream:** after a validation or oversized-packet error, `ReadPacket` does not drain the announced bytes, so packet alignment is lost. The caller must close the connection.
- **Write errors:** when the underlying writer fails part way, `WritePacket` returns the error without reporting how many octets were sent. A partial packet may be on the wire.
- **Receive allocation:** `ReadPacket` allocates the payload from the declared length before the payload arrives, up to `ReaderConfig.MaxPacketLength` minus the header (65531 octets by default). Lower the limit for connections that do not need full-size packets.
- **Allocation per packet:** each `ReadPacket` and `WritePacket` allocates one packet-sized buffer; there is no buffer reuse.
- **Reserved octet:** it is ignored on input, so a non-zero value from a peer is accepted and is not reported.
- **Minimum payload:** payloads below 3 octets are rejected on encode and on decode, following the RFC 1006 minimum packet length.

## Platforms

The package has no platform-specific code. CI runs tests on Linux only. `make cross` compiles for linux/amd64, linux/arm64, linux/arm, darwin/arm64 and windows/amd64 without running tests there.

## Timing

Nothing here is a real-time guarantee. Latency depends on the transport and the runtime, and the benchmarks in `bench_test.go` are not a timing specification.
