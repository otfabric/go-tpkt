// SPDX-License-Identifier: MIT

package tpkt

import (
	"fmt"
	"io"
)

// Reader reads TPKT-framed payloads from an underlying io.Reader.
//
// It is safe for use with any streaming source such as a net.Conn. Reader is
// not safe for concurrent use from multiple goroutines without external
// synchronization.
type Reader struct {
	r            io.Reader
	maxFrameSize int // maximum allowed total TPKT length, including header
}

// ReaderOption configures a Reader.
type ReaderOption func(*Reader)

// WithMaxFrameSize sets an upper bound on the total TPKT packet size (header
// plus payload) that the Reader will accept.
//
// Values less than or equal to zero leave the default in place. Values greater
// than zero but smaller than MinPacketLength are clamped up to MinPacketLength.
func WithMaxFrameSize(n int) ReaderOption {
	return func(r *Reader) {
		if n <= 0 {
			return
		}
		if n < MinPacketLength {
			n = MinPacketLength
		}
		r.maxFrameSize = n
	}
}

// NewReader constructs a Reader over r.
//
// By default it accepts packets up to the RFC 1006 maximum. WithMaxFrameSize
// can be used to impose a stricter bound.
func NewReader(r io.Reader, opts ...ReaderOption) *Reader {
	rd := &Reader{
		r:            r,
		maxFrameSize: rfcMaxPacketLength,
	}
	for _, opt := range opts {
		opt(rd)
	}
	return rd
}

// ReadFrame reads the next TPKT frame from the stream and returns its payload.
//
// It returns io.EOF when called at a frame boundary and the underlying reader
// has no more data. If the stream ends in the middle of a header or payload,
// it returns an error wrapping io.ErrUnexpectedEOF.
func (r *Reader) ReadFrame() ([]byte, error) {
	var header [HeaderLength]byte
	if err := readFullHeader(r.r, header[:]); err != nil {
		return nil, err
	}

	totalLen, err := decodeHeader(header[:])
	if err != nil {
		return nil, fmt.Errorf("read tpkt header: %w", err)
	}

	if totalLen > r.maxFrameSize {
		return nil, fmt.Errorf("read tpkt header: total length=%d exceeds maxFrameSize=%d: %w", totalLen, r.maxFrameSize, ErrFrameTooLarge)
	}

	payloadLen := totalLen - HeaderLength
	payload := make([]byte, payloadLen)
	if err := readFullPayload(r.r, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func readFullHeader(r io.Reader, header []byte) error {
	n, err := io.ReadFull(r, header)
	if err != nil {
		// If nothing was read and we hit EOF, surface EOF so callers can
		// distinguish between a clean end-of-stream and a truncated header.
		if err == io.EOF && n == 0 {
			return io.EOF
		}
		return fmt.Errorf("read tpkt header: %w", err)
	}
	return nil
}

func readFullPayload(r io.Reader, payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	if _, err := io.ReadFull(r, payload); err != nil {
		return fmt.Errorf("read tpkt payload: %w", err)
	}
	return nil
}
