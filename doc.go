// SPDX-License-Identifier: MIT

// Package tpkt implements TPKT framing for RFC 1006 and RFC 2126 transport profiles.
//
// TPKT is a simple header + payload packetization scheme used to carry ISO
// transport protocol data units (TPDUs) over a TCP byte stream. Each packet
// starts with a 4-byte header:
//
//   - byte 0: version (always 3)
//   - byte 1: reserved (written as 0; ignored on input per RFC 2126 §6.10)
//   - bytes 2–3: total packet length in octets, big-endian, including header
//
// This package provides TPKT framing compliant with RFC 1006 §6 and
// RFC 2126 §4.3/§6.10. It does not implement the broader ITOT / Class 0/2
// transport profile of RFC 2126, and it does not interpret TPDU contents.
//
// Out of scope:
//
//   - COTP or CR/CC/DT/ED/EA/DR/DC TPDU semantics
//   - Dual TCP expedited-data connections
//   - TSAP / NSAPA addressing or ISO session/presentation
//   - S7comm, MMS, or other application protocols
//   - TCP dial, listen, or port binding
//
// The library enforces the RFC 1006 minimum total packet length of 7 octets
// (opaque payload at least 3 octets) but treats the TPDU as opaque. Higher
// layers (e.g. COTP) validate TPDU structure and semantics.
//
// Public entry points:
//
//   - EncodePacket / DecodePacket for complete buffers
//   - Reader.ReadPacket / Writer.WritePacket for streams
//
// DecodePacket requires exactly one complete TPKT and rejects trailing bytes.
// ReadPacket returns io.EOF only at a clean packet boundary; EOF after any
// byte of a packet has been consumed is reported as io.ErrUnexpectedEOF.
//
// A Reader and a Writer may be used concurrently on the same full-duplex
// connection. Individual Reader and Writer values are not safe for concurrent
// use by multiple goroutines.
package tpkt
