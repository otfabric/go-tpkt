// SPDX-License-Identifier: MIT

package tpkt

import (
	"fmt"
	"io"
)

// Writer writes opaque payloads as TPKT packets to an underlying io.Writer.
//
// A Reader and a Writer may be used concurrently when they wrap the same
// full-duplex connection. An individual Writer is not safe for concurrent use
// by multiple goroutines without external synchronization.
type Writer struct {
	w io.Writer
}

// NewWriter constructs a Writer over w. w must be non-nil.
func NewWriter(w io.Writer) (*Writer, error) {
	if w == nil {
		return nil, ErrNilWriter
	}
	return &Writer{w: w}, nil
}

// WritePacket encodes payload as a TPKT and writes it in full.
//
// The reserved octet is always written as zero. Short writes from the
// underlying writer are retried until the complete packet is sent or an error
// occurs. A zero-byte write with a nil error yields io.ErrShortWrite.
func (w *Writer) WritePacket(payload []byte) error {
	pkt, err := EncodePacket(payload)
	if err != nil {
		return err
	}
	total := len(pkt)

	written := 0
	for written < total {
		n, err := w.w.Write(pkt[written:])
		if n > 0 {
			written += n
		}
		if err != nil {
			return fmt.Errorf("write tpkt: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("write tpkt: %w", io.ErrShortWrite)
		}
	}

	return nil
}
