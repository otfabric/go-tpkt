# go-tpkt

[![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/otfabric/go-tpkt.svg)](https://pkg.go.dev/github.com/otfabric/go-tpkt)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/otfabric/go-tpkt/actions/workflows/ci.yml/badge.svg)](https://github.com/otfabric/go-tpkt/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/otfabric/go-tpkt/graph/badge.svg)](https://codecov.io/gh/otfabric/go-tpkt)
[![Release](https://img.shields.io/github/v/release/otfabric/go-tpkt?label=release)](https://github.com/otfabric/go-tpkt/releases)

`go-tpkt` is a small, idiomatic Go library that implements TPKT framing for
[RFC 1006](https://datatracker.ietf.org/doc/html/rfc1006) and
[RFC 2126](https://datatracker.ietf.org/doc/html/rfc2126) transport profiles.

TPKT is a simple header + payload packet format used to carry ISO transport
protocol data units (TPDUs) over a TCP byte stream. This package focuses only
on TPKT framing and validation; it does not interpret TPDUs or implement the
broader ITOT / Class 0/2 transport profile of RFC 2126.

### Table of contents

- [Scope](#scope)
- [Install](#install)
- [Getting started](#getting-started)
  - [Encode and decode a single packet](#encode-and-decode-a-single-packet)
  - [Streaming reader](#streaming-reader)
  - [Streaming writer](#streaming-writer)
- [Relation to higher-level protocols](#relation-to-higher-level-protocols)
- [License](#license)
- [API reference](API.md)

### Scope

- **In scope**:
  - TPKT header construction and parsing (RFC 1006 §6, RFC 2126 §4.3/§6.10)
  - `EncodePacket` / `DecodePacket` for complete buffers
  - Streaming `Reader` / `Writer` over `io.Reader` / `io.Writer`
  - Validation of version and length (min=7, max=65535)
  - Configurable receive size limit via `ReaderConfig`
  - Reserved octet: written as 0; ignored on input

- **Out of scope**:
  - COTP / CR/CC/DT TPDU parsing
  - Dual TCP expedited-data connections
  - TSAP / NSAPA addressing or ISO session/presentation
  - S7comm, MMS, IEC 61850, or any application protocol logic
  - TCP dial, listen, or port binding

`go-tpkt` is a foundation for higher-level stacks such as COTP over RFC 1006.

### Install

```bash
go get github.com/otfabric/go-tpkt@v1.0.0
```

Requires Go 1.23 or newer.

### Getting started

The examples below cover the most common entry points. For the complete public
API — wire format, errors, EOF semantics, and the requirements matrix — see
**[API.md](API.md)**.

#### Encode and decode a single packet

```go
package main

import (
	"log"

	"github.com/otfabric/go-tpkt"
)

func main() {
	payload := []byte{0x02, 0xf0, 0x80}

	pkt, err := tpkt.EncodePacket(payload)
	if err != nil {
		log.Fatalf("encode: %v", err)
	}

	decoded, err := tpkt.DecodePacket(pkt)
	if err != nil {
		log.Fatalf("decode: %v", err)
	}

	_ = decoded // use TPDU bytes in a higher-level protocol
}
```

#### Streaming reader

```go
package main

import (
	"bytes"
	"log"

	"github.com/otfabric/go-tpkt"
)

func main() {
	pkt, _ := tpkt.EncodePacket([]byte{0x01, 0x02, 0x03})

	r, err := tpkt.NewReader(bytes.NewReader(pkt), tpkt.ReaderConfig{})
	if err != nil {
		log.Fatal(err)
	}
	payload, err := r.ReadPacket()
	if err != nil {
		log.Fatalf("read: %v", err)
	}

	_ = payload // TPDU bytes for a higher-level protocol
}
```

For `net.Conn` usage, EOF semantics, and `ReaderConfig.MaxPacketLength`, see
[Stream API](API.md#stream-api) in API.md.

#### Streaming writer

```go
package main

import (
	"bytes"
	"log"

	"github.com/otfabric/go-tpkt"
)

func main() {
	var buf bytes.Buffer
	w, err := tpkt.NewWriter(&buf)
	if err != nil {
		log.Fatal(err)
	}

	payload := []byte{0x01, 0x02, 0x03}
	if err := w.WritePacket(payload); err != nil {
		log.Fatalf("write: %v", err)
	}

	_ = buf.Bytes() // complete TPKT packet on the wire
}
```

### Relation to higher-level protocols

This package intentionally stops at TPKT framing. Protocols such as COTP,
S7comm, and MMS can be implemented on top of the opaque payloads read and
written through this library.

See [Usage patterns](API.md#usage-patterns) in API.md.

## License

This project is licensed under the MIT License. See [LICENSE](./LICENSE).
