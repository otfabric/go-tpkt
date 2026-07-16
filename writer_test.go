// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestWriterWriteFrame(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	expectedPkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	var buf bytes.Buffer
	w := NewWriter(&buf)

	n, err := w.WriteFrame(payload)
	if err != nil {
		t.Fatalf("WriteFrame() error = %v", err)
	}
	if n != len(expectedPkt) {
		t.Fatalf("WriteFrame() wrote %d bytes, want %d", n, len(expectedPkt))
	}
	if got := buf.Bytes(); !bytes.Equal(got, expectedPkt) {
		t.Fatalf("written bytes mismatch: got %v want %v", got, expectedPkt)
	}
}

type errWriter struct {
	err error
}

func (e *errWriter) Write(p []byte) (int, error) {
	return 0, e.err
}

func TestWriterUnderlyingError(t *testing.T) {
	wantErr := errors.New("boom")
	w := NewWriter(&errWriter{err: wantErr})

	if _, err := w.WriteFrame([]byte{0x01, 0x02, 0x03}); err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("WriteFrame() error = %v, want to wrap %v", err, wantErr)
	}
}

type partialWriter struct {
	writes int
}

func (p *partialWriter) Write(b []byte) (int, error) {
	p.writes++
	if p.writes == 1 {
		// First call: short write without error.
		n := len(b) / 2
		if n == 0 && len(b) > 0 {
			n = 1
		}
		return n, nil
	}
	// Second call: fail.
	return 0, io.ErrClosedPipe
}

func TestWriterShortWrite(t *testing.T) {
	pw := &partialWriter{}
	w := NewWriter(pw)

	n, err := w.WriteFrame([]byte{0x01, 0x02, 0x03, 0x04})
	if err == nil {
		t.Fatal("WriteFrame() expected error, got nil")
	}
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("WriteFrame() error = %v, want io.ErrClosedPipe", err)
	}
	if n <= 0 {
		t.Fatalf("WriteFrame() wrote %d bytes, want > 0 before error", n)
	}
}

type zeroWriter struct{}

func (z *zeroWriter) Write(p []byte) (int, error) {
	return 0, nil
}

func TestWriterPureShortWrite(t *testing.T) {
	zw := &zeroWriter{}
	w := NewWriter(zw)

	if _, err := w.WriteFrame([]byte{0x01, 0x02, 0x03}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("WriteFrame() error = %v, want io.ErrShortWrite", err)
	}
}
