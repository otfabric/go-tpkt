package tpkt

import (
	"encoding/binary"
	"fmt"
)

// Frame represents a single TPKT-framed TPDU.
//
// The Payload is treated as an opaque sequence of bytes by this package; higher
// level protocols (e.g. COTP, S7, MMS) are expected to interpret it. A Frame
// value by itself does not guarantee RFC 1006 validity: payloads that are too
// short or too large will be rejected when marshaled or encoded.
type Frame struct {
	Payload []byte
}

// Len reports HeaderLength + len(Payload), i.e. the total length the frame
// would occupy on the wire if encodable.
func (f Frame) Len() int {
	return HeaderLength + len(f.Payload)
}

// MarshalBinary encodes the Frame into a complete TPKT packet.
//
// The returned slice is a newly allocated buffer. If the encoded packet would
// be smaller than MinPacketLength or larger than MaxPacketLength, an error is
// returned.
func (f Frame) MarshalBinary() ([]byte, error) {
	return Encode(f.Payload)
}

// Encode builds a complete TPKT packet from the provided payload.
//
// The returned slice contains the 4-byte TPKT header followed by the payload.
// The header is constructed according to RFC 1006 section 6.
func Encode(payload []byte) ([]byte, error) {
	totalLen := HeaderLength + len(payload)

	if totalLen < rfcMinPacketLength {
		return nil, fmt.Errorf("encode tpkt: total length %d < minimum %d: %w", totalLen, rfcMinPacketLength, ErrInvalidLength)
	}
	if totalLen > rfcMaxPacketLength {
		return nil, fmt.Errorf("encode tpkt: total length %d > maximum %d: %w", totalLen, rfcMaxPacketLength, ErrFrameTooLarge)
	}

	buf := make([]byte, totalLen)
	buf[0] = Version
	buf[1] = 0 // reserved
	binary.BigEndian.PutUint16(buf[2:4], uint16(totalLen))
	copy(buf[HeaderLength:], payload)

	return buf, nil
}

// Decode validates a complete TPKT packet and returns only the payload.
//
// The returned slice aliases pkt; callers must copy it if they need to retain
// it independently.
func Decode(pkt []byte) ([]byte, error) {
	return parse(pkt)
}

// Parse validates a complete TPKT packet and returns a Frame.
//
// The returned Frame.Payload aliases pkt; callers must copy it if they need to
// retain it independently.
func Parse(pkt []byte) (Frame, error) {
	payload, err := parse(pkt)
	if err != nil {
		return Frame{}, err
	}
	return Frame{Payload: payload}, nil
}

// parse performs common validation logic for Decode and Parse.
//
// It returns a slice view of the payload, which aliases pkt.
func parse(pkt []byte) ([]byte, error) {
	if len(pkt) < HeaderLength {
		return nil, fmt.Errorf("decode tpkt: have %d bytes, need at least header: %w", len(pkt), ErrTooShort)
	}

	totalLen, err := decodeHeader(pkt[:HeaderLength])
	if err != nil {
		return nil, fmt.Errorf("decode tpkt: %w", err)
	}

	if totalLen != len(pkt) {
		return nil, fmt.Errorf("decode tpkt: declared length=%d, actual=%d: %w", totalLen, len(pkt), ErrLengthMismatch)
	}

	payloadLen := totalLen - HeaderLength
	return pkt[HeaderLength : HeaderLength+payloadLen], nil
}

// decodeHeader performs structural validation of the first 4 bytes of a TPKT
// header and returns the declared total packet length in octets.
func decodeHeader(hdr []byte) (int, error) {
	if len(hdr) < HeaderLength {
		return 0, fmt.Errorf("tpkt header: have %d bytes, need %d: %w", len(hdr), HeaderLength, ErrTooShort)
	}

	version := hdr[0]
	if version != Version {
		return 0, fmt.Errorf("tpkt header: version=%d: %w", version, ErrInvalidVersion)
	}

	reserved := hdr[1]
	if reserved != 0 {
		return 0, fmt.Errorf("tpkt header: reserved=%d: %w", reserved, ErrInvalidReserved)
	}

	totalLen := int(binary.BigEndian.Uint16(hdr[2:4]))
	if totalLen < rfcMinPacketLength {
		return 0, fmt.Errorf("tpkt header: total length=%d: %w", totalLen, ErrInvalidLength)
	}

	return totalLen, nil
}
