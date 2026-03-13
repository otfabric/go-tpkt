# go-tpkt

[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/otfabric/go-tpkt)](https://goreportcard.com/report/github.com/otfabric/go-tpkt)
[![CI](https://github.com/otfabric/go-tpkt/actions/workflows/test.yml/badge.svg)](https://github.com/otfabric/go-tpkt/actions/workflows/test.yml)
[![Codecov](https://codecov.io/github/otfabric/go-tpkt/graph/badge.svg)](https://app.codecov.io/github/otfabric/go-tpkt)
[![Release](https://img.shields.io/github/v/release/otfabric/go-tpkt?label=release)](https://github.com/otfabric/go-tpkt/releases)

`go-tpkt` is a small, idiomatic Go library that implements the TPKT packet
framing defined in [RFC 1006](https://datatracker.ietf.org/doc/html/rfc1006).

TPKT is a simple header + payload packet format used to carry ISO transport
protocol data units (TPDUs) over a TCP byte stream. This package focuses only
on TPKT framing and validation; it does not interpret or implement any
higher-level transport or application protocols.

### Scope

- **In scope**:
  - TPKT header construction and parsing
  - Encode/decode helpers for complete packets
  - Streaming `Reader` and `Writer` over `io.Reader` / `io.Writer`
  - Strict validation of version, reserved byte, and length fields
    (RFC 1006 `packet length` min=7, max=65535)
  - Protection against oversized frames

- **Out of scope**:
  - COTP / CR/CC/DT TPDU parsing
  - TSAP addressing or ISO session/presentation
  - S7comm, MMS, IEC 61850, or any application protocol logic
  - TCP listener/server management

`go-tpkt` is intended as a foundation for higher-level stacks such as COTP,
S7comm, or MMS over RFC 1006.

### Install

```bash
go get github.com/otfabric/go-tpkt
```

Requires Go 1.22 or newer.

### Basic usage

#### Encode and decode a single packet

```go
package main

import (
	"log"

	"github.com/otfabric/go-tpkt"
)

func main() {
	payload := []byte{0x02, 0xf0, 0x80}

	pkt, err := tpkt.Encode(payload)
	if err != nil {
		log.Fatalf("encode: %v", err)
	}

	decoded, err := tpkt.Decode(pkt)
	if err != nil {
		log.Fatalf("decode: %v", err)
	}

	_ = decoded // use TPDU bytes in a higher-level protocol
}
```

#### Streaming reader

```go
// conn is a net.Conn established to a peer speaking RFC 1006 TPKT.
// r := tpkt.NewReader(conn)

// payload, err := r.ReadFrame()
// if err != nil {
//     // handle EOF, malformed frames, etc.
// }
// // payload now contains a complete TPDU as bytes.
```

#### Streaming writer

```go
// conn is a net.Conn
// w := tpkt.NewWriter(conn)
// _, err := w.WriteFrame(payload)
// if err != nil {
//     // handle write or framing error
// }
```

### Relation to higher-level protocols

This package intentionally stops at TPKT framing. Protocols such as COTP,
S7comm, and MMS can be implemented on top of the payloads read and written
through this library without any coupling to their semantics.

### Size limits, validation, and ownership

- The library enforces the RFC 1006 minimum packet size of 7 octets
  (4-byte TPKT header + 3-byte minimum TPDU), so payloads must be at least
  3 bytes long to be encodable.
- `tpkt.MinPacketLength` and `tpkt.MaxPacketLength` expose the protocol
  bounds (7 and 65535) for callers that want to size buffers or apply their
  own checks.
- `Encode`, `Decode`, and `Parse` validate packets against protocol structure
  only (version, reserved, length, buffer consistency); they do not apply any
  additional caller-configurable maximum frame size.
- `Reader` applies the same structural checks and also enforces a configurable
  maximum total packet size via `WithMaxFrameSize`, clamping values below
  `tpkt.MinPacketLength` up to that minimum.
- `Decode` and `Parse` return payload slices that alias the input buffer; if
  you need to retain or mutate them independently, copy the data first.
- `Reader.ReadFrame` returns a newly allocated payload slice that is safe to
  mutate without affecting the underlying stream buffer.
- `Reader` and `Writer` are not safe for concurrent use from multiple
  goroutines without external synchronization.

