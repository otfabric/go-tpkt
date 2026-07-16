// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"errors"
	"io"
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
