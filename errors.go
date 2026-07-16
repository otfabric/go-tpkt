// SPDX-License-Identifier: MIT

package tpkt

import "errors"

// Version is the TPKT protocol version supported by this package.
// RFC 1006 defines this value as 3.
const Version byte = 3

// HeaderLength is the length in octets of the fixed TPKT header.
const HeaderLength = 4

const (
	// rfcMinPacketLength is the minimum legal TPKT packet length according to
	// RFC 1006 section 6 (7 octets: 4-byte header + 3-byte minimum TPDU).
	rfcMinPacketLength = 7

	// rfcMaxPacketLength is the maximum legal TPKT packet length according to
	// the 16-bit length field in RFC 1006 section 6.
	rfcMaxPacketLength = 65535
)

// Exported protocol size bounds for callers that want to configure buffers or
// validation in terms of on-the-wire TPKT sizes.
const (
	MinPacketLength = rfcMinPacketLength
	MaxPacketLength = rfcMaxPacketLength
)

// Exported sentinel errors. Callers should use errors.Is to classify failures.
var (
	// ErrTooShort indicates that a buffer is shorter than the 4-byte TPKT header.
	ErrTooShort = errors.New("tpkt: packet too short")
	// ErrInvalidVersion indicates a header with a version byte other than 3.
	ErrInvalidVersion = errors.New("tpkt: invalid version")
	// ErrInvalidReserved indicates a header with a non-zero reserved byte.
	ErrInvalidReserved = errors.New("tpkt: invalid reserved byte")
	// ErrInvalidLength indicates that the declared TPKT length is outside
	// the legal range or otherwise structurally invalid.
	ErrInvalidLength = errors.New("tpkt: invalid length field")
	// ErrLengthMismatch indicates that the declared TPKT length does not
	// match the actual size of the buffer.
	ErrLengthMismatch = errors.New("tpkt: length field does not match buffer size")
	// ErrFrameTooLarge indicates that a frame exceeds a configured or
	// protocol-defined maximum size.
	ErrFrameTooLarge = errors.New("tpkt: frame exceeds maximum allowed size")
)
