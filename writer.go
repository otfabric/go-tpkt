// SPDX-License-Identifier: MIT

package tpkt

import (
	"fmt"
	"io"
)

// Writer writes opaque payloads as TPKT packets to an underlying io.Writer.
//
// Construct a Writer with NewWriter. The zero value has no stream and is not
// usable: WritePacket on a zero-value or nil Writer returns ErrNilWriter.
//
// A Writer whose WritePacket failed after it had written one or more octets
// of a packet stays failed: the stream is no longer aligned on a packet
// boundary, and every later WritePacket returns an error that matches
// ErrStreamMisaligned without writing to the stream.
//
// A Reader and a Writer may be used concurrently when they wrap the same
// full-duplex connection. An individual Writer is not safe for concurrent use
// by multiple goroutines without external synchronization.
type Writer struct {
	w io.Writer

	// misaligned is nil until a WritePacket fails after it wrote one or more
	// octets of a packet. It then matches ErrStreamMisaligned, wraps the
	// error of that call, and is what every later WritePacket returns.
	misaligned error
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
// occurs. A zero-byte write with a nil error yields io.ErrShortWrite. A
// write that reports a negative count or more octets than it was given breaks
// the io.Writer contract; WritePacket then returns an error and the Writer
// stays failed as after a partial write.
//
// A payload outside [MinPayloadLength, MaxPayloadLength] is refused with
// ErrPayloadTooShort or ErrPayloadTooLarge before the Writer looks at its
// stream: nothing is written, and the error is the same in every state of
// the Writer.
//
// An error of the underlying writer, or io.ErrShortWrite, before any octet
// of the packet was written leaves the Writer usable, and WritePacket may be
// called again.
//
// The same error after one or more octets of the packet were written leaves
// an incomplete packet on the stream. The call returns the error, wrapped.
// The stream is no longer aligned on a packet boundary (the peer would read
// the start of the next packet as the rest of this one), so the Writer
// records the error. Every later WritePacket writes nothing and returns an
// error that matches ErrStreamMisaligned and wraps the recorded one, so
// errors.Is and errors.As still find the cause. The caller closes the
// connection.
//
// WritePacket on a zero-value or nil Writer returns ErrNilWriter.
func (w *Writer) WritePacket(payload []byte) error {
	pkt, err := EncodePacket(payload)
	if err != nil {
		return err
	}
	// Checked after the payload, so a payload error is reported as before.
	if w == nil || w.w == nil {
		return ErrNilWriter
	}
	if w.misaligned != nil {
		return w.misaligned
	}
	total := len(pkt)

	written := 0
	for written < total {
		n, err := w.w.Write(pkt[written:])
		if n < 0 || n > total-written {
			// The writer broke the io.Writer contract, so how much of the
			// packet reached the stream is unknown: treat the stream as
			// misaligned.
			err = fmt.Errorf("write tpkt: %w: %d of %d octets", errInvalidWriteCount, n, total-written)
			w.misaligned = fmt.Errorf("%w: %w", ErrStreamMisaligned, err)
			return err
		}
		if n > 0 {
			written += n
		}
		if err != nil {
			return w.fail(written, fmt.Errorf("write tpkt: %w", err))
		}
		if n == 0 {
			return w.fail(written, fmt.Errorf("write tpkt: %w", io.ErrShortWrite))
		}
	}

	return nil
}

// fail returns err. When one or more octets of the packet were written, it
// also records err as the error that left the stream inside a packet; every
// later WritePacket returns the recorded error.
func (w *Writer) fail(written int, err error) error {
	if written > 0 {
		w.misaligned = fmt.Errorf("%w: %w", ErrStreamMisaligned, err)
	}
	return err
}
