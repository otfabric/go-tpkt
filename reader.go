// SPDX-License-Identifier: MIT

package tpkt

import (
	"errors"
	"fmt"
	"io"
)

// Reader reads TPKT packets from an underlying io.Reader.
//
// A Reader and a Writer may be used concurrently when they wrap the same
// full-duplex connection. An individual Reader is not safe for concurrent use
// by multiple goroutines without external synchronization.
type Reader struct {
	r               io.Reader
	maxPacketLength int
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
// After a validation or oversized-packet error, the stream is unusable; the
// caller must close the connection. Remaining announced bytes are not drained.
func (r *Reader) ReadPacket() ([]byte, error) {
	var header [HeaderLength]byte
	if err := readFullHeader(r.r, header[:]); err != nil {
		return nil, err
	}

	totalLen, err := decodeHeader(header[:])
	if err != nil {
		return nil, fmt.Errorf("read tpkt header: %w", err)
	}

	if totalLen > r.maxPacketLength {
		return nil, fmt.Errorf("read tpkt header: total length=%d exceeds maxPacketLength=%d: %w",
			totalLen, r.maxPacketLength, ErrPacketTooLarge)
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
		// Clean end-of-stream only when no header byte was consumed.
		// io.ReadFull already maps a partial header + EOF to ErrUnexpectedEOF.
		if errors.Is(err, io.EOF) && n == 0 {
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
