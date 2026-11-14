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

func TestNewReaderValidation(t *testing.T) {
	if _, err := NewReader(nil, ReaderConfig{}); !errors.Is(err, ErrNilReader) {
		t.Fatalf("got %v, want ErrNilReader", err)
	}
	r := bytes.NewReader(nil)
	if _, err := NewReader(r, ReaderConfig{MaxPacketLength: 1}); !errors.Is(err, ErrInvalidMaxPacketLength) {
		t.Fatalf("max=1: got %v, want ErrInvalidMaxPacketLength", err)
	}
	if _, err := NewReader(r, ReaderConfig{MaxPacketLength: MaxPacketLength + 1}); !errors.Is(err, ErrInvalidMaxPacketLength) {
		t.Fatalf("max too large: got %v, want ErrInvalidMaxPacketLength", err)
	}
	rd, err := NewReader(r, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if rd.maxPacketLength != MaxPacketLength {
		t.Fatalf("default max = %d, want %d", rd.maxPacketLength, MaxPacketLength)
	}
	rd, err = NewReader(r, ReaderConfig{MaxPacketLength: MinPacketLength})
	if err != nil {
		t.Fatal(err)
	}
	if rd.maxPacketLength != MinPacketLength {
		t.Fatalf("max = %d, want %d", rd.maxPacketLength, MinPacketLength)
	}
}

func TestReaderSinglePacket(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := EncodePacket(payload)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(bytes.NewReader(pkt), ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadPacket()
	if err != nil {
		t.Fatalf("ReadPacket: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %v want %v", got, payload)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, io.EOF) {
		t.Fatalf("second ReadPacket: %v, want io.EOF", err)
	}
}

func TestReaderEmptyStreamEOF(t *testing.T) {
	r, err := NewReader(bytes.NewReader(nil), ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, io.EOF) {
		t.Fatalf("got %v, want io.EOF", err)
	}
}

func TestReaderMultiplePackets(t *testing.T) {
	p1, p2 := []byte("one"), []byte("two")
	pkt1, _ := EncodePacket(p1)
	pkt2, _ := EncodePacket(p2)
	var buf bytes.Buffer
	buf.Write(pkt1)
	buf.Write(pkt2)

	r, err := NewReader(&buf, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got1, err := r.ReadPacket()
	if err != nil || !bytes.Equal(got1, p1) {
		t.Fatalf("packet1: got %v err %v", got1, err)
	}
	got2, err := r.ReadPacket()
	if err != nil || !bytes.Equal(got2, p2) {
		t.Fatalf("packet2: got %v err %v", got2, err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, io.EOF) {
		t.Fatalf("third: %v, want EOF", err)
	}
}

func TestReaderTruncatedHeader(t *testing.T) {
	pkt, _ := EncodePacket([]byte{0x01, 0x02, 0x03})
	r, err := NewReader(bytes.NewReader(pkt[:2]), ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want ErrUnexpectedEOF", err)
	}
}

func TestReaderHeaderThenClose(t *testing.T) {
	// Complete valid header announcing payload, then EOF — truncated packet.
	header := []byte{Version, 0x00, 0x00, 0x07}
	r, err := NewReader(bytes.NewReader(header), ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want ErrUnexpectedEOF (not clean EOF)", err)
	}
}

func TestReadFullPayloadEmpty(t *testing.T) {
	// Declared total length is always ≥ MinPacketLength, so ReadPacket never
	// requests a zero-length payload; cover the helper guard directly.
	if err := readFullPayload(bytes.NewReader(nil), nil); err != nil {
		t.Fatalf("empty payload: %v", err)
	}
}

func TestReaderTruncatedPayload(t *testing.T) {
	pkt, _ := EncodePacket([]byte{0x01, 0x02, 0x03, 0x04})
	r, err := NewReader(bytes.NewReader(pkt[:HeaderLength+1]), ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want ErrUnexpectedEOF", err)
	}
}

func TestReaderMaxPacketLength(t *testing.T) {
	payload := []byte("this is a reasonably sized payload")
	pkt, _ := EncodePacket(payload)
	r, err := NewReader(bytes.NewReader(pkt), ReaderConfig{MaxPacketLength: HeaderLength + 8})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, ErrPacketTooLarge) {
		t.Fatalf("got %v, want ErrPacketTooLarge", err)
	}
}

func TestReaderIgnoresReserved(t *testing.T) {
	pkt := []byte{Version, 0xff, 0x00, 0x07, 0x01, 0x02, 0x03}
	r, err := NewReader(bytes.NewReader(pkt), ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte{0x01, 0x02, 0x03}) {
		t.Fatalf("got %v", got)
	}
}

func TestReaderHeaderValidationErrors(t *testing.T) {
	base, _ := EncodePacket([]byte{0x01, 0x02, 0x03})
	tests := []struct {
		name    string
		mutate  func([]byte) []byte
		wantErr error
	}{
		{
			name: "invalid version",
			mutate: func(b []byte) []byte {
				cp := append([]byte(nil), b...)
				cp[0] = 0x02
				return cp
			},
			wantErr: ErrInvalidVersion,
		},
		{
			name: "invalid length below minimum",
			mutate: func(b []byte) []byte {
				cp := append([]byte(nil), b...)
				cp[2], cp[3] = 0x00, 0x06
				return cp
			},
			wantErr: ErrInvalidPacketLength,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReader(bytes.NewReader(tt.mutate(base)), ReaderConfig{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.ReadPacket(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestReaderReadPacketAllocatesIndependentPayload(t *testing.T) {
	pkt, _ := EncodePacket([]byte{0x01, 0x02, 0x03})
	orig := append([]byte(nil), pkt...)
	r, err := NewReader(bytes.NewReader(pkt), ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	got[0] ^= 0xff
	if !bytes.Equal(pkt, orig) {
		t.Fatal("expected original packet buffer unchanged")
	}
}

// chunkReader is a hostile io.Reader that returns data in scheduled chunks.
type chunkReader struct {
	data   []byte
	chunks []int
	pos    int
	ci     int
	eofN   bool // if true, last non-empty read returns (n, io.EOF)
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if c.pos >= len(c.data) {
		return 0, io.EOF
	}
	n := len(p)
	if c.ci < len(c.chunks) {
		n = c.chunks[c.ci]
		c.ci++
		if n <= 0 {
			n = 1
		}
	}
	remain := len(c.data) - c.pos
	if n > remain {
		n = remain
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p[:n], c.data[c.pos:c.pos+n])
	c.pos += n
	if c.pos >= len(c.data) && c.eofN {
		return n, io.EOF
	}
	return n, nil
}

func TestReaderFragmentation(t *testing.T) {
	payload := []byte{0x0a, 0x0b, 0x0c, 0x0d, 0x0e}
	pkt, err := EncodePacket(payload)
	if err != nil {
		t.Fatal(err)
	}

	schedules := [][]int{
		{1, 1, 1, 1, 1, 1, 1, 1, 1},
		{2, 2, 3},
		{4, 3},
		{7},
		{1, 3, 2, 1},
		{HeaderLength}, // header then rest in default chunks
	}

	for _, chunks := range schedules {
		t.Run("", func(t *testing.T) {
			cr := &chunkReader{data: pkt, chunks: chunks}
			r, err := NewReader(cr, ReaderConfig{})
			if err != nil {
				t.Fatal(err)
			}
			got, err := r.ReadPacket()
			if err != nil {
				t.Fatalf("chunks %v: %v", chunks, err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatalf("chunks %v: got %v want %v", chunks, got, payload)
			}
		})
	}
}

func TestReaderChunkFinalByteWithEOF(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, _ := EncodePacket(payload)
	// Return data one byte at a time; last byte comes with io.EOF.
	chunks := make([]int, len(pkt))
	for i := range chunks {
		chunks[i] = 1
	}
	cr := &chunkReader{data: pkt, chunks: chunks, eofN: true}
	r, err := NewReader(cr, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %v want %v", got, payload)
	}
}

// stickyErrReader returns (0, err) once, then serves data normally.
// Verifies ReadPacket surfaces a non-EOF zero-byte read error.
type stickyErrReader struct {
	data []byte
	pos  int
	err  error
	once bool
}

func (s *stickyErrReader) Read(p []byte) (int, error) {
	if !s.once {
		s.once = true
		return 0, s.err
	}
	if s.pos >= len(s.data) {
		return 0, io.EOF
	}
	n := copy(p, s.data[s.pos:])
	s.pos += n
	return n, nil
}

func TestReaderZeroByteCustomError(t *testing.T) {
	pkt, err := EncodePacket([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("temporary read failure")
	r, err := NewReader(&stickyErrReader{data: pkt, err: wantErr}, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want wrap of %v", err, wantErr)
	}
}

func TestReaderCoalescedPackets(t *testing.T) {
	p1, p2 := []byte{0x01, 0x02, 0x03}, []byte{0x04, 0x05, 0x06}
	pkt1, _ := EncodePacket(p1)
	pkt2, _ := EncodePacket(p2)
	combined := append(append([]byte(nil), pkt1...), pkt2...)

	// Deliver everything in one Read.
	cr := &chunkReader{data: combined, chunks: []int{len(combined)}}
	r, err := NewReader(cr, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got1, err := r.ReadPacket()
	if err != nil || !bytes.Equal(got1, p1) {
		t.Fatalf("first: %v %v", got1, err)
	}
	got2, err := r.ReadPacket()
	if err != nil || !bytes.Equal(got2, p2) {
		t.Fatalf("second: %v %v", got2, err)
	}
}

func TestReaderPacketAPlusPartialHeaderB(t *testing.T) {
	p1 := []byte{0x01, 0x02, 0x03}
	pkt1, _ := EncodePacket(p1)
	pkt2, _ := EncodePacket([]byte{0x0a, 0x0b, 0x0c})
	data := append(append([]byte(nil), pkt1...), pkt2[:2]...)

	r, err := NewReader(bytes.NewReader(data), ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadPacket()
	if err != nil || !bytes.Equal(got, p1) {
		t.Fatalf("first packet: %v %v", got, err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("partial next header: %v, want ErrUnexpectedEOF", err)
	}
}

func TestReaderReservedThenAligned(t *testing.T) {
	p1 := []byte{0x01, 0x02, 0x03}
	p2 := []byte{0x0a, 0x0b, 0x0c}
	pkt1 := []byte{Version, 0x80, 0x00, 0x07, 0x01, 0x02, 0x03}
	pkt2, _ := EncodePacket(p2)
	var buf bytes.Buffer
	buf.Write(pkt1)
	buf.Write(pkt2)

	r, err := NewReader(&buf, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got1, err := r.ReadPacket()
	if err != nil || !bytes.Equal(got1, p1) {
		t.Fatalf("first: %v %v", got1, err)
	}
	got2, err := r.ReadPacket()
	if err != nil || !bytes.Equal(got2, p2) {
		t.Fatalf("second: %v %v", got2, err)
	}
}

// scriptStep is one step of a scriptReader: the octets it hands out, then
// the error it returns.
type scriptStep struct {
	data []byte
	err  error
}

// scriptReader is an io.Reader that plays its steps in order. After the last
// step it returns io.EOF. It counts the Read calls it receives.
type scriptReader struct {
	steps []scriptStep
	calls int
}

func (s *scriptReader) Read(p []byte) (int, error) {
	s.calls++
	for len(s.steps) > 0 {
		st := &s.steps[0]
		if len(st.data) > 0 {
			n := copy(p, st.data)
			st.data = st.data[n:]
			return n, nil
		}
		err := st.err
		s.steps = s.steps[1:]
		if err != nil {
			return 0, err
		}
	}
	return 0, io.EOF
}

// TestReaderEOFClassification ends the stream after each prefix of one
// packet. Only the empty prefix is a clean end and returns io.EOF itself;
// every other prefix is io.ErrUnexpectedEOF and never matches io.EOF, also
// when the stream wraps its EOF in an error of its own.
func TestReaderEOFClassification(t *testing.T) {
	pkt, err := EncodePacket([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatal(err)
	}
	ends := []struct {
		name string
		err  error
	}{
		{name: "plain EOF", err: io.EOF},
		{name: "wrapped EOF", err: fmt.Errorf("stream: %w", io.EOF)},
	}
	for _, end := range ends {
		for n := range len(pkt) {
			t.Run(fmt.Sprintf("%s after %d octets", end.name, n), func(t *testing.T) {
				src := &scriptReader{steps: []scriptStep{{data: bytes.Clone(pkt[:n]), err: end.err}}}
				r, err := NewReader(src, ReaderConfig{})
				if err != nil {
					t.Fatal(err)
				}
				payload, err := r.ReadPacket()
				if payload != nil {
					t.Fatalf("payload of %d octets from a truncated stream", len(payload))
				}
				if n == 0 {
					if err != io.EOF {
						t.Fatalf("got %v, want io.EOF itself", err)
					}
					return
				}
				if !errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
					t.Fatalf("got %v, want io.ErrUnexpectedEOF and no match of io.EOF", err)
				}
			})
		}
	}
}

// TestReaderStreamErrorKeepsIdentity: an error of the stream other than EOF
// is returned wrapped, whatever part of the packet arrived before it. A
// caller that classifies a read deadline with errors.Is still finds it, and
// it is not reported as an end of the stream.
func TestReaderStreamErrorKeepsIdentity(t *testing.T) {
	pkt, err := EncodePacket([]byte{0x01, 0x02, 0x03, 0x04, 0x05})
	if err != nil {
		t.Fatal(err)
	}
	for n := range len(pkt) {
		t.Run(fmt.Sprintf("deadline after %d octets", n), func(t *testing.T) {
			src := &scriptReader{steps: []scriptStep{{data: bytes.Clone(pkt[:n]), err: os.ErrDeadlineExceeded}}}
			r, err := NewReader(src, ReaderConfig{})
			if err != nil {
				t.Fatal(err)
			}
			payload, err := r.ReadPacket()
			if payload != nil {
				t.Fatalf("payload of %d octets with an error", len(payload))
			}
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Fatalf("got %v, want a wrap of os.ErrDeadlineExceeded", err)
			}
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("got %v, a stream error must not match an end of stream", err)
			}
		})
	}
}

// TestReaderUsableAfterErrorAtPacketBoundary: an error before any octet of a
// packet was consumed leaves the stream on a packet boundary and the Reader
// usable. The error may come more than once, before the first packet or
// between two packets; the next ReadPacket returns the packet.
func TestReaderUsableAfterErrorAtPacketBoundary(t *testing.T) {
	first, second := []byte{0x01, 0x02, 0x03}, []byte{0x0a, 0x0b, 0x0c, 0x0d}
	pkt1, err := EncodePacket(first)
	if err != nil {
		t.Fatal(err)
	}
	pkt2, err := EncodePacket(second)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		err  error // what the stream returns on the packet boundary
		want error // what ReadPacket returns for it
	}{
		{name: "deadline", err: os.ErrDeadlineExceeded, want: os.ErrDeadlineExceeded},
		{name: "EOF", err: io.EOF, want: io.EOF},
		{name: "wrapped EOF", err: fmt.Errorf("stream: %w", io.EOF), want: io.EOF},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := &scriptReader{steps: []scriptStep{
				{err: tc.err},
				{err: tc.err},
				{data: bytes.Clone(pkt1), err: tc.err},
				{data: bytes.Clone(pkt2)},
			}}
			r, err := NewReader(src, ReaderConfig{})
			if err != nil {
				t.Fatal(err)
			}
			readError := func(when string) {
				t.Helper()
				payload, err := r.ReadPacket()
				if !errors.Is(err, tc.want) || payload != nil {
					t.Fatalf("%s: got %v and %d octets, want %v and no payload", when, err, len(payload), tc.want)
				}
				if errors.Is(err, ErrStreamMisaligned) {
					t.Fatalf("%s: got %v, an error on a packet boundary must leave the Reader usable", when, err)
				}
			}
			readPacket := func(when string, want []byte) {
				t.Helper()
				got, err := r.ReadPacket()
				if err != nil {
					t.Fatalf("%s: %v", when, err)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("%s: got %x want %x", when, got, want)
				}
			}
			readError("first call")
			readError("second call")
			readPacket("first packet", first)
			readError("between the packets")
			readPacket("second packet", second)
			if _, err := r.ReadPacket(); err != io.EOF {
				t.Fatalf("behind the packets: got %v, want io.EOF", err)
			}
		})
	}
}

// expectMisaligned calls ReadPacket twice on a Reader that failed inside a
// packet. Each call must return no payload and an error that matches
// ErrStreamMisaligned and cause.
func expectMisaligned(t *testing.T, r *Reader, cause error) {
	t.Helper()
	for i := range 2 {
		payload, err := r.ReadPacket()
		if payload != nil {
			t.Fatalf("later call %d returned %x, which was not sent as a packet (err=%v)", i+1, payload, err)
		}
		if !errors.Is(err, ErrStreamMisaligned) {
			t.Fatalf("later call %d: got %v, want ErrStreamMisaligned", i+1, err)
		}
		if !errors.Is(err, cause) {
			t.Fatalf("later call %d: got %v, which does not wrap the cause %v", i+1, err, cause)
		}
	}
}

// TestReaderStaysFailedAfterStreamErrorInsidePacket: a read that fails after
// one or more octets of a packet were consumed, in the header or in the
// payload, leaves the Reader failed. That call returns the error of the
// stream. Every later call returns an error that matches ErrStreamMisaligned
// and the cause, and reads nothing, although the stream would deliver the
// rest of the packet and another packet.
func TestReaderStaysFailedAfterStreamErrorInsidePacket(t *testing.T) {
	pkt, err := EncodePacket([]byte{0x01, 0x02, 0x03, 0x04, 0x05})
	if err != nil {
		t.Fatal(err)
	}
	next, err := EncodePacket([]byte{0x0a, 0x0b, 0x0c})
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n < len(pkt); n++ {
		t.Run(fmt.Sprintf("deadline after %d octets", n), func(t *testing.T) {
			src := &scriptReader{steps: []scriptStep{
				{data: bytes.Clone(pkt[:n]), err: os.ErrDeadlineExceeded},
				{data: bytes.Clone(pkt[n:])},
				{data: bytes.Clone(next)},
			}}
			r, err := NewReader(src, ReaderConfig{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.ReadPacket(); !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Fatalf("first call: got %v, want a wrap of os.ErrDeadlineExceeded", err)
			}
			calls := src.calls
			expectMisaligned(t, r, os.ErrDeadlineExceeded)
			if src.calls != calls {
				t.Fatalf("%d reads from the stream after the error, want none", src.calls-calls)
			}
		})
	}
}

// TestReaderRetryInsidePacketReturnsNoPayload: the payload of the packet on
// the stream is itself a well-formed TPKT. A deadline expires behind the
// header of the outer packet. A second ReadPacket must not take the payload
// octets for a packet of their own.
func TestReaderRetryInsidePacketReturnsNoPayload(t *testing.T) {
	inner, err := EncodePacket([]byte{0xaa, 0xbb, 0xcc})
	if err != nil {
		t.Fatal(err)
	}
	outer, err := EncodePacket(inner)
	if err != nil {
		t.Fatal(err)
	}
	src := &scriptReader{steps: []scriptStep{
		{data: bytes.Clone(outer[:HeaderLength]), err: os.ErrDeadlineExceeded},
		{data: bytes.Clone(outer[HeaderLength:])},
	}}
	r, err := NewReader(src, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("first call: got %v, want a wrap of os.ErrDeadlineExceeded", err)
	}
	expectMisaligned(t, r, os.ErrDeadlineExceeded)
}

// TestReaderStaysFailedAfterRefusedOrTruncatedPacket: a refused header and
// a stream that ends inside a packet leave the Reader failed as well. The
// later calls wrap the first error, read nothing, and never report a clean
// end of the stream.
func TestReaderStaysFailedAfterRefusedOrTruncatedPacket(t *testing.T) {
	good, err := EncodePacket([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		stream []byte
		limit  int
		want   error
	}{
		{name: "version", stream: append([]byte{0x02, 0x00, 0x00, 0x07, 0x01, 0x02, 0x03}, good...), want: ErrInvalidVersion},
		{name: "declared length", stream: append([]byte{Version, 0x00, 0x00, 0x06, 0x01, 0x02}, good...), want: ErrInvalidPacketLength},
		{name: "receive limit", stream: append(rawPacket(8), good...), limit: MinPacketLength, want: ErrPacketTooLarge},
		{name: "end inside the header", stream: good[:2], want: io.ErrUnexpectedEOF},
		{name: "end behind the header", stream: good[:HeaderLength], want: io.ErrUnexpectedEOF},
		{name: "end inside the payload", stream: good[:len(good)-1], want: io.ErrUnexpectedEOF},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := bytes.NewReader(tc.stream)
			r, err := NewReader(src, ReaderConfig{MaxPacketLength: tc.limit})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.ReadPacket(); !errors.Is(err, tc.want) {
				t.Fatalf("first call: got %v, want %v", err, tc.want)
			}
			left := src.Len()
			expectMisaligned(t, r, tc.want)
			if _, err := r.ReadPacket(); errors.Is(err, io.EOF) {
				t.Fatalf("got %v, a stream that stopped inside a packet must not end with io.EOF", err)
			}
			if src.Len() != left {
				t.Fatalf("%d octets read from the stream after the error, want none", left-src.Len())
			}
		})
	}
}

// TestReaderZeroValueAndNil: a Reader that NewReader did not build has no
// stream. ReadPacket reports that with ErrNilReader.
func TestReaderZeroValueAndNil(t *testing.T) {
	readers := []struct {
		name string
		r    *Reader
	}{
		{name: "zero value", r: &Reader{}},
		{name: "nil receiver", r: nil},
	}
	for _, tc := range readers {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := tc.r.ReadPacket()
			if !errors.Is(err, ErrNilReader) {
				t.Fatalf("got %v, want ErrNilReader", err)
			}
			if payload != nil {
				t.Fatalf("payload of %d octets with an error", len(payload))
			}
		})
	}
}
