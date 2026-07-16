// SPDX-License-Identifier: MIT

package tpkt

import "errors"

// Version is the TPKT protocol version supported by this package.
// RFC 1006 and RFC 2126 both use version 3 for the TPKT header.
const Version byte = 3

// HeaderLength is the length in octets of the fixed TPKT header.
const HeaderLength = 4

// Protocol size bounds. Lengths are total TPKT sizes on the wire unless noted.
const (
	// MinPacketLength is the minimum legal TPKT total packet length
	// (RFC 1006 §6: 4-byte header + 3-byte minimum TPDU).
	MinPacketLength = 7

	// MaxPacketLength is the maximum legal TPKT total packet length
	// (16-bit length field in RFC 1006 §6 / RFC 2126 §4.3).
	MaxPacketLength = 65535

	// MinPayloadLength is the minimum opaque TPDU payload length.
	MinPayloadLength = MinPacketLength - HeaderLength

	// MaxPayloadLength is the maximum opaque TPDU payload length.
	MaxPayloadLength = MaxPacketLength - HeaderLength
)

// Exported sentinel errors. Callers should use errors.Is to classify failures.
var (
	// ErrNilReader indicates NewReader was called with a nil io.Reader.
	ErrNilReader = errors.New("tpkt: nil reader")
	// ErrNilWriter indicates NewWriter was called with a nil io.Writer.
	ErrNilWriter = errors.New("tpkt: nil writer")
	// ErrTooShort indicates that a buffer is shorter than the 4-byte TPKT header.
	ErrTooShort = errors.New("tpkt: packet too short")
	// ErrInvalidVersion indicates a header with a version byte other than 3.
	ErrInvalidVersion = errors.New("tpkt: invalid version")
	// ErrInvalidPacketLength indicates that a declared or computed TPKT length
	// is outside the legal range [MinPacketLength, MaxPacketLength].
	ErrInvalidPacketLength = errors.New("tpkt: invalid packet length")
	// ErrInvalidMaxPacketLength indicates that ReaderConfig.MaxPacketLength
	// is outside [MinPacketLength, MaxPacketLength] (zero means default).
	ErrInvalidMaxPacketLength = errors.New("tpkt: invalid maximum packet length")
	// ErrLengthMismatch indicates that the declared TPKT length does not
	// match the actual size of the input buffer.
	ErrLengthMismatch = errors.New("tpkt: declared length does not match input")
	// ErrPacketTooLarge indicates that a packet exceeds the reader's
	// configured maximum total TPKT size.
	ErrPacketTooLarge = errors.New("tpkt: packet exceeds configured maximum")
	// ErrPayloadTooShort indicates that an encode payload is shorter than
	// MinPayloadLength (3 octets).
	ErrPayloadTooShort = errors.New("tpkt: payload shorter than minimum")
	// ErrPayloadTooLarge indicates that an encode payload exceeds
	// MaxPayloadLength (65531 octets).
	ErrPayloadTooLarge = errors.New("tpkt: payload exceeds maximum")
)
