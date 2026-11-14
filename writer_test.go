// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
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

// TestWriterZeroValueAndNil: a Writer that NewWriter did not build has no
// stream. WritePacket reports that with ErrNilWriter.
func TestWriterZeroValueAndNil(t *testing.T) {
	writers := []struct {
		name string
		w    *Writer
	}{
		{name: "zero value", w: &Writer{}},
		{name: "nil receiver", w: nil},
	}
	for _, tc := range writers {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.w.WritePacket([]byte{0x01, 0x02, 0x03}); !errors.Is(err, ErrNilWriter) {
				t.Fatalf("got %v, want ErrNilWriter", err)
			}
		})
	}
}

// writeStep is one Write of a scriptWriter: the octets it accepts, at most
// the ones it is given, and the error it returns with them.
type writeStep struct {
	n   int
	err error
}

// scriptWriter is an io.Writer that plays its steps in order and accepts
// every Write in full after the last step. It keeps what it accepted and
// counts the Write calls it receives.
type scriptWriter struct {
	buf   bytes.Buffer
	steps []writeStep
	calls int
}

func (s *scriptWriter) Write(p []byte) (int, error) {
	s.calls++
	if len(s.steps) == 0 {
		return s.buf.Write(p)
	}
	st := s.steps[0]
	s.steps = s.steps[1:]
	n := min(st.n, len(p))
	s.buf.Write(p[:n])
	return n, st.err
}

// TestWriterUsableAfterErrorBeforeAnyOctet: a Write that fails, or makes no
// progress, before any octet of a packet was written leaves the stream on a
// packet boundary and the Writer usable. The next WritePacket puts one whole
// packet on the stream.
func TestWriterUsableAfterErrorBeforeAnyOctet(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	want, err := EncodePacket(payload)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		step writeStep
		want error
	}{
		{name: "deadline", step: writeStep{err: os.ErrDeadlineExceeded}, want: os.ErrDeadlineExceeded},
		{name: "no progress", step: writeStep{}, want: io.ErrShortWrite},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sw := &scriptWriter{steps: []writeStep{tc.step, tc.step}}
			w, err := NewWriter(sw)
			if err != nil {
				t.Fatal(err)
			}
			for i := range 2 {
				err := w.WritePacket(payload)
				if !errors.Is(err, tc.want) {
					t.Fatalf("call %d: got %v, want %v", i+1, err, tc.want)
				}
				if errors.Is(err, ErrStreamMisaligned) {
					t.Fatalf("call %d: got %v, an error before any octet must leave the Writer usable", i+1, err)
				}
			}
			if err := w.WritePacket(payload); err != nil {
				t.Fatalf("third call: %v", err)
			}
			if !bytes.Equal(sw.buf.Bytes(), want) {
				t.Fatalf("stream holds %x, want one packet %x", sw.buf.Bytes(), want)
			}
		})
	}
}

// TestWriterStaysFailedAfterErrorInsidePacket: a write that fails, or stops
// making progress, after one or more octets of a packet were written leaves
// the Writer failed. That call returns the error. Every later call returns
// an error that matches ErrStreamMisaligned and the cause, and writes
// nothing behind the incomplete packet. A payload-size error is still
// reported as such.
func TestWriterStaysFailedAfterErrorInsidePacket(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	pkt, err := EncodePacket(payload)
	if err != nil {
		t.Fatal(err)
	}
	failures := []struct {
		name  string
		steps func(k int) []writeStep // k octets are accepted before the failure
		most  int                     // the largest k the failure can follow
		want  error
	}{
		{
			// The error can come with the last octet of the packet too.
			name:  "error with the octets",
			steps: func(k int) []writeStep { return []writeStep{{n: k, err: os.ErrDeadlineExceeded}} },
			most:  len(pkt),
			want:  os.ErrDeadlineExceeded,
		},
		{
			name:  "error behind the octets",
			steps: func(k int) []writeStep { return []writeStep{{n: k}, {err: os.ErrDeadlineExceeded}} },
			most:  len(pkt) - 1,
			want:  os.ErrDeadlineExceeded,
		},
		{
			name:  "no progress behind the octets",
			steps: func(k int) []writeStep { return []writeStep{{n: k}, {}} },
			most:  len(pkt) - 1,
			want:  io.ErrShortWrite,
		},
	}
	for _, f := range failures {
		for k := 1; k <= f.most; k++ {
			t.Run(fmt.Sprintf("%s/%d octets", f.name, k), func(t *testing.T) {
				sw := &scriptWriter{steps: f.steps(k)}
				w, err := NewWriter(sw)
				if err != nil {
					t.Fatal(err)
				}
				if err := w.WritePacket(payload); !errors.Is(err, f.want) {
					t.Fatalf("first call: got %v, want %v", err, f.want)
				}
				if !bytes.Equal(sw.buf.Bytes(), pkt[:k]) {
					t.Fatalf("stream holds %x, want the first %d octets of the packet", sw.buf.Bytes(), k)
				}
				calls := sw.calls
				for i := range 2 {
					err := w.WritePacket([]byte{0x0a, 0x0b, 0x0c})
					if !errors.Is(err, ErrStreamMisaligned) {
						t.Fatalf("later call %d: got %v, want ErrStreamMisaligned", i+1, err)
					}
					if !errors.Is(err, f.want) {
						t.Fatalf("later call %d: got %v, which does not wrap the cause %v", i+1, err, f.want)
					}
				}
				if err := w.WritePacket([]byte{0x01}); !errors.Is(err, ErrPayloadTooShort) {
					t.Fatalf("short payload: got %v, want ErrPayloadTooShort", err)
				}
				if sw.calls != calls || sw.buf.Len() != k {
					t.Fatalf("%d writes and %d octets on the stream after the error, want none",
						sw.calls-calls, sw.buf.Len()-k)
				}
			})
		}
	}
}
