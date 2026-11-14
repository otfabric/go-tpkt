// SPDX-License-Identifier: MIT

package tpkt

import (
	"errors"
	"fmt"
	"io"
)

// Reader reads TPKT packets from an underlying io.Reader.
//
// Construct a Reader with NewReader. The zero value has no stream and is not
// usable: ReadPacket on a zero-value or nil Reader returns ErrNilReader.
//
// A Reader whose ReadPacket failed after it had consumed one or more octets
// of a packet stays failed: the stream is no longer aligned on a packet
// boundary, and every later ReadPacket returns an error that matches
// ErrStreamMisaligned without reading from the stream.
//
// A Reader and a Writer may be used concurrently when they wrap the same
// full-duplex connection. An individual Reader is not safe for concurrent use
// by multiple goroutines without external synchronization.
type Reader struct {
	r               io.Reader
	maxPacketLength int

	// misaligned is nil until a ReadPacket fails after it consumed one or
	// more octets of a packet. It then matches ErrStreamMisaligned, wraps
	// the error of that call, and is what every later ReadPacket returns.
	misaligned error
}

// ReaderConfig configures a Reader.
//
// MaxPacketLength is the maximum accepted total TPKT size (header included).
// Zero means MaxPacketLength (65535). Values outside
// [MinPacketLength, MaxPacketLength] are rejected — never clamped.
type ReaderConfig struct {
	MaxPacketLength int
}

// NewReader constructs a Reader over r.
//
// An empty ReaderConfig{} uses the protocol maximum. r must be non-nil.
func NewReader(r io.Reader, cfg ReaderConfig) (*Reader, error) {
	if r == nil {
		return nil, ErrNilReader
	}

	maxLen := cfg.MaxPacketLength
	if maxLen == 0 {
		maxLen = MaxPacketLength
	}
	if maxLen < MinPacketLength || maxLen > MaxPacketLength {
		return nil, fmt.Errorf("%w: got %d, expected %d..%d or 0 for default",
			ErrInvalidMaxPacketLength, cfg.MaxPacketLength, MinPacketLength, MaxPacketLength)
	}

	return &Reader{
		r:               r,
		maxPacketLength: maxLen,
	}, nil
}

// ReadPacket reads the next TPKT from the stream and returns its payload.
//
// It returns io.EOF only when called at a packet boundary and no header byte
// is available. Any EOF after one or more bytes of a packet have been
// consumed — including immediately after a valid header — is reported as
// io.ErrUnexpectedEOF.
//
// An error before any octet of a packet was consumed leaves the Reader
// usable, and ReadPacket may be called again. That is io.EOF, and an error
// of the stream on a packet boundary (a read deadline on an idle connection,
// for example), which is returned wrapped.
//
// An error after one or more octets of a packet were consumed leaves the
// stream inside that packet. That is a validation or oversized-packet error
// (the announced octets are not drained), an end of the stream, and an
// error of the stream such as a read deadline, which is returned wrapped.
// The octets consumed are discarded and the stream is no longer aligned on
// a packet boundary, so the Reader records the error. Every later
// ReadPacket reads nothing and returns an error that matches
// ErrStreamMisaligned and wraps the recorded one, so errors.Is and
// errors.As still find the cause. The caller closes the connection.
//
// ReadPacket on a zero-value or nil Reader returns ErrNilReader.
func (r *Reader) ReadPacket() ([]byte, error) {
	if r == nil || r.r == nil {
		return nil, ErrNilReader
	}
	if r.misaligned != nil {
		return nil, r.misaligned
	}

	var header [HeaderLength]byte
	n, err := readFullHeader(r.r, header[:])
	if err != nil {
		if n > 0 {
			return nil, r.fail(err)
		}
		// No octet of a packet was consumed: the stream is still on a
		// packet boundary and the caller may read again.
		return nil, err
	}

	totalLen, err := decodeHeader(header[:])
	if err != nil {
		return nil, r.fail(fmt.Errorf("read tpkt header: %w", err))
	}

	if totalLen > r.maxPacketLength {
		return nil, r.fail(fmt.Errorf("read tpkt header: total length=%d exceeds maxPacketLength=%d: %w",
			totalLen, r.maxPacketLength, ErrPacketTooLarge))
	}

	payloadLen := totalLen - HeaderLength
	payload := make([]byte, payloadLen)
	if err := readFullPayload(r.r, payload); err != nil {
		return nil, r.fail(err)
	}

	return payload, nil
}

// fail records err as the error that left the stream inside a packet and
// returns err. Every later ReadPacket returns the recorded error.
func (r *Reader) fail(err error) error {
	r.misaligned = fmt.Errorf("%w: %w", ErrStreamMisaligned, err)
	return err
}

// readFullHeader fills header and returns the number of octets it consumed.
func readFullHeader(r io.Reader, header []byte) (int, error) {
	n, err := io.ReadFull(r, header)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// Clean end-of-stream only when no header byte was consumed.
			if n == 0 {
				return 0, io.EOF
			}
			// A partial header is a truncated packet. io.ReadFull maps a
			// bare io.EOF to io.ErrUnexpectedEOF, but compares with ==, so
			// an EOF that the stream wraps arrives here unchanged.
			err = io.ErrUnexpectedEOF
		}
		return n, fmt.Errorf("read tpkt header: %w", err)
	}
	return n, nil
}

func readFullPayload(r io.Reader, payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	_, err := io.ReadFull(r, payload)
	if err == nil {
		return nil
	}
	// After a valid header, missing payload is a truncated packet — never a
	// clean end-of-stream.
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	return fmt.Errorf("read tpkt payload: %w", err)
}
