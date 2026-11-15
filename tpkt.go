// SPDX-License-Identifier: MIT

package tpkt

import (
	"encoding/binary"
	"fmt"
)

// EncodePacket builds exactly one TPKT from the provided opaque payload.
//
// The returned slice is newly allocated and contains the 4-byte TPKT header
// followed by the payload. The reserved octet is always written as zero
// (RFC 2126 §4.3). Payload length must be in [MinPayloadLength, MaxPayloadLength].
func EncodePacket(payload []byte) ([]byte, error) {
	if err := checkPayloadLength(len(payload)); err != nil {
		return nil, err
	}
	buf := make([]byte, HeaderLength+len(payload))
	encodeInto(buf, payload)
	return buf, nil
}

// checkPayloadLength refuses a payload length outside
// [MinPayloadLength, MaxPayloadLength].
func checkPayloadLength(n int) error {
	if n < MinPayloadLength {
		return fmt.Errorf("encode tpkt: payload length %d < minimum %d: %w", n, MinPayloadLength, ErrPayloadTooShort)
	}
	if n > MaxPayloadLength {
		return fmt.Errorf("encode tpkt: payload length %d > maximum %d: %w", n, MaxPayloadLength, ErrPayloadTooLarge)
	}
	return nil
}

// encodeInto writes the header and payload into pkt, which is exactly
// HeaderLength+len(payload) octets long. The caller has checked the payload
// length with checkPayloadLength.
func encodeInto(pkt, payload []byte) {
	totalLen := len(pkt)
	// Bound totalLen itself so the uint16 conversion below is visibly in
	// range.
	if totalLen < MinPacketLength || totalLen > MaxPacketLength {
		panic("tpkt: encodeInto: packet length out of range")
	}
	pkt[0] = Version
	pkt[1] = 0 // reserved
	binary.BigEndian.PutUint16(pkt[2:4], uint16(totalLen))
	copy(pkt[HeaderLength:], payload)
}

// DecodePacket validates exactly one complete TPKT and returns its payload.
//
// Trailing bytes after the declared packet length are rejected with
// ErrLengthMismatch. The reserved octet is ignored on input as recommended by
// RFC 2126 §6.10. The returned slice aliases packet; callers must copy it if
// they need to retain it independently. Its capacity ends at the end of the
// packet, so appending to it copies and never writes into the buffer behind
// packet.
func DecodePacket(packet []byte) ([]byte, error) {
	if len(packet) < HeaderLength {
		return nil, fmt.Errorf("decode tpkt: have %d bytes, need at least header: %w", len(packet), ErrTooShort)
	}

	totalLen, err := decodeHeader(packet[:HeaderLength])
	if err != nil {
		return nil, fmt.Errorf("decode tpkt: %w", err)
	}

	if totalLen != len(packet) {
		return nil, fmt.Errorf("decode tpkt: declared length=%d, actual=%d: %w", totalLen, len(packet), ErrLengthMismatch)
	}

	// The full slice expression ends the capacity at the packet: packet may be
	// a prefix of a larger buffer of the caller.
	return packet[HeaderLength:totalLen:totalLen], nil
}

// decodeHeader validates the first 4 bytes of a TPKT header and returns the
// declared total packet length in octets. The reserved octet is ignored.
func decodeHeader(hdr []byte) (int, error) {
	if len(hdr) < HeaderLength {
		return 0, fmt.Errorf("tpkt header: have %d bytes, need %d: %w", len(hdr), HeaderLength, ErrTooShort)
	}

	if hdr[0] != Version {
		return 0, fmt.Errorf("tpkt header: version=%d: %w", hdr[0], ErrInvalidVersion)
	}

	// Reserved octet (hdr[1]) is ignored on input per RFC 2126 §6.10.

	totalLen := int(binary.BigEndian.Uint16(hdr[2:4]))
	if totalLen < MinPacketLength || totalLen > MaxPacketLength {
		return 0, fmt.Errorf("tpkt header: total length=%d: %w", totalLen, ErrInvalidPacketLength)
	}

	return totalLen, nil
}
