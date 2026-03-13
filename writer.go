package tpkt

import (
	"fmt"
	"io"
)

// Writer writes payloads as TPKT-framed packets to an underlying io.Writer.
// It currently exposes only the WriteFrame helper; callers that wish to send
// a Frame should pass f.Payload explicitly. Writer is not safe for concurrent
// use from multiple goroutines without external synchronization.
type Writer struct {
	w io.Writer
}

// NewWriter constructs a Writer over w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteFrame encodes payload as a TPKT packet and writes it in full.
//
// It returns the total number of octets written (header plus payload). If the
// underlying writer performs a short write or returns an error, WriteFrame
// returns a non-nil error.
func (w *Writer) WriteFrame(payload []byte) (int, error) {
	pkt, err := Encode(payload)
	if err != nil {
		return 0, err
	}
	total := len(pkt)

	written := 0
	for written < total {
		n, err := w.w.Write(pkt[written:])
		if n > 0 {
			written += n
		}
		if err != nil {
			return written, fmt.Errorf("write tpkt: %w", err)
		}
		if n == 0 {
			return written, fmt.Errorf("write tpkt: %w", io.ErrShortWrite)
		}
	}

	return written, nil
}
