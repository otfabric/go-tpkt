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
	totalLen := HeaderLength + len(payload)

	if len(payload) < MinPayloadLength {
		return nil, fmt.Errorf("encode tpkt: payload length %d < minimum %d: %w", len(payload), MinPayloadLength, ErrPayloadTooShort)
	}
	if len(payload) > MaxPayloadLength {
		return nil, fmt.Errorf("encode tpkt: payload length %d > maximum %d: %w", len(payload), MaxPayloadLength, ErrPayloadTooLarge)
	}

	buf := make([]byte, totalLen)
	buf[0] = Version
	buf[1] = 0 // reserved
	binary.BigEndian.PutUint16(buf[2:4], uint16(totalLen))
	copy(buf[HeaderLength:], payload)

	return buf, nil
}

// DecodePacket validates exactly one complete TPKT and returns its payload.
//
// Trailing bytes after the declared packet length are rejected with
// ErrLengthMismatch. The reserved octet is ignored on input as recommended by
// RFC 2126 §6.10. The returned slice aliases packet; callers must copy it if
// they need to retain it independently.
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

	payloadLen := totalLen - HeaderLength
	return packet[HeaderLength : HeaderLength+payloadLen], nil
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
