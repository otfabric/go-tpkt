// SPDX-License-Identifier: MIT

// Package tpkt implements TPKT framing as defined by RFC 1006.
//
// TPKT is a simple header + payload packetization scheme used to carry ISO
// transport protocol data units (TPDUs) over a TCP byte stream. Each packet
// starts with a 4-byte header:
//
//   - byte 0: version (always 3 for this version of the protocol)
//   - byte 1: reserved (always 0)
//   - bytes 2–3: total packet length in octets, big-endian, including header
//
// This package is intentionally limited to TPKT framing only. It does not
// interpret or construct TPDUs and it does not implement:
//
//   - COTP or CR/CC/DT TPDU semantics
//   - TSAP addressing or ISO session/presentation
//   - S7comm, MMS, or other application protocols
//
// Callers are expected to build higher-level protocol stacks on top of the
// payload bytes returned by Decode, Parse, and Reader.ReadFrame, and to send
// payload bytes using Writer.WriteFrame.
//
// The implementation follows RFC 1006 section 6 (Packet Format) and enforces:
//
//   - version == 3
//   - reserved == 0
//   - total length field within the legal range [7, 65535]
//   - consistency between the declared length and the actual buffer size
//
// A total length of 7 bytes corresponds to a 4-byte TPKT header plus a
// 3-byte minimum TPDU for transport class 0. Frames declaring a length
// smaller than 7 (including a 4-byte header with no TPDU bytes) are rejected.
//
// Validation and payload ownership:
//
//   - Decode and Parse validate packets against protocol structure only
//     (version, reserved, length, and buffer consistency). They do not apply
//     any additional caller-configurable maximum frame size.
//   - Reader.ReadFrame applies the same structural checks and also enforces a
//     configurable maximum total packet size before allocating payload memory.
//   - Decode and Parse: the returned payload aliases the input buffer.
//   - Reader.ReadFrame: the returned slice is a newly allocated payload.
//
// Reader and Writer are not safe for concurrent use from multiple goroutines
// without external synchronization.
package tpkt
