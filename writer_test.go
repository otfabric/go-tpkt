// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestNewWriterNil(t *testing.T) {
	if _, err := NewWriter(nil); !errors.Is(err, ErrNilWriter) {
		t.Fatalf("got %v, want ErrNilWriter", err)
	}
}

func TestWriterWritePacket(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	expected, err := EncodePacket(payload)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w, err := NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WritePacket(payload); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Fatalf("wire mismatch: got %v want %v", buf.Bytes(), expected)
	}
	if buf.Bytes()[1] != 0 {
		t.Fatal("reserved must be 0 on wire")
	}
}

type errWriter struct{ err error }

func (e *errWriter) Write([]byte) (int, error) { return 0, e.err }

func TestWriterUnderlyingError(t *testing.T) {
	wantErr := errors.New("boom")
	w, err := NewWriter(&errWriter{err: wantErr})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WritePacket([]byte{0x01, 0x02, 0x03}); err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want wrap of %v", err, wantErr)
	}
}

type partialWriter struct {
	writes int
	limit  int
}

func (p *partialWriter) Write(b []byte) (int, error) {
	p.writes++
	if p.limit > 0 && len(b) > p.limit {
		return p.limit, nil
	}
	if p.writes == 1 {
		n := len(b) / 2
		if n == 0 && len(b) > 0 {
			n = 1
		}
		return n, nil
	}
	return 0, io.ErrClosedPipe
}

func TestWriterShortWriteThenError(t *testing.T) {
	w, err := NewWriter(&partialWriter{})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WritePacket([]byte{0x01, 0x02, 0x03, 0x04}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("got %v, want ErrClosedPipe", err)
	}
}

type chunkWriter struct {
	buf   bytes.Buffer
	chunk int
}

func (c *chunkWriter) Write(b []byte) (int, error) {
	n := c.chunk
	if n <= 0 {
		n = 1
	}
	if n > len(b) {
		n = len(b)
	}
	return c.buf.Write(b[:n])
}

func TestWriterShortWritesComplete(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	expected, _ := EncodePacket(payload)
	cw := &chunkWriter{chunk: 2}
	w, err := NewWriter(cw)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WritePacket(payload); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cw.buf.Bytes(), expected) {
		t.Fatalf("got %v want %v", cw.buf.Bytes(), expected)
	}
}

type zeroWriter struct{}

func (z *zeroWriter) Write([]byte) (int, error) { return 0, nil }

func TestWriterZeroProgress(t *testing.T) {
	w, err := NewWriter(&zeroWriter{})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WritePacket([]byte{0x01, 0x02, 0x03}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("got %v, want ErrShortWrite", err)
	}
}

func TestWriterRejectsShortPayload(t *testing.T) {
	w, err := NewWriter(&bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WritePacket([]byte{0x01}); !errors.Is(err, ErrPayloadTooShort) {
		t.Fatalf("got %v, want ErrPayloadTooShort", err)
	}
}
