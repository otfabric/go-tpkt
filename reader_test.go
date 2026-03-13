package tpkt

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestReaderSingleFrame(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	r := NewReader(bytes.NewReader(pkt))

	got, err := r.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload mismatch: got %v want %v", got, payload)
	}

	// Second read at end-of-stream should yield io.EOF.
	if _, err := r.ReadFrame(); !errors.Is(err, io.EOF) {
		t.Fatalf("second ReadFrame() error = %v, want io.EOF", err)
	}
}

func TestReaderEmptyStreamEOF(t *testing.T) {
	r := NewReader(bytes.NewReader(nil))

	if _, err := r.ReadFrame(); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadFrame() on empty stream error = %v, want io.EOF", err)
	}
}

func TestReaderMultipleFrames(t *testing.T) {
	p1 := []byte("one")
	p2 := []byte("two")

	pkt1, err := Encode(p1)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	pkt2, err := Encode(p2)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	var buf bytes.Buffer
	buf.Write(pkt1)
	buf.Write(pkt2)

	r := NewReader(&buf)

	got1, err := r.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() #1 error = %v", err)
	}
	if !bytes.Equal(got1, p1) {
		t.Fatalf("frame1 payload mismatch: got %v want %v", got1, p1)
	}

	got2, err := r.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() #2 error = %v", err)
	}
	if !bytes.Equal(got2, p2) {
		t.Fatalf("frame2 payload mismatch: got %v want %v", got2, p2)
	}

	if _, err := r.ReadFrame(); !errors.Is(err, io.EOF) {
		t.Fatalf("third ReadFrame() error = %v, want io.EOF", err)
	}
}

func TestReaderTruncatedHeader(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Provide only part of the header.
	truncated := pkt[:2]
	r := NewReader(bytes.NewReader(truncated))

	if _, err := r.ReadFrame(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("ReadFrame() error = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestReaderTruncatedPayload(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Provide full header but only part of payload.
	truncated := pkt[:HeaderLength+1]
	r := NewReader(bytes.NewReader(truncated))

	if _, err := r.ReadFrame(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("ReadFrame() error = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestReaderMaxFrameSize(t *testing.T) {
	// Use a modest payload to exceed a very small maxFrameSize.
	payload := []byte("this is a reasonably sized payload")
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	r := NewReader(bytes.NewReader(pkt), WithMaxFrameSize(HeaderLength+8))

	if _, err := r.ReadFrame(); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("ReadFrame() error = %v, want ErrFrameTooLarge", err)
	}
}

func TestReaderWithMaxFrameSizeSemantics(t *testing.T) {
	// WithMaxFrameSize(0) should leave default in place and accept a valid frame.
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	r0 := NewReader(bytes.NewReader(pkt), WithMaxFrameSize(0))
	if _, err := r0.ReadFrame(); err != nil {
		t.Fatalf("ReadFrame() with maxFrameSize=0 error = %v, want nil", err)
	}

	// WithMaxFrameSize(1) should be clamped to MinPacketLength, so the same
	// valid frame is still accepted.
	r1 := NewReader(bytes.NewReader(pkt), WithMaxFrameSize(1))
	if _, err := r1.ReadFrame(); err != nil {
		t.Fatalf("ReadFrame() with maxFrameSize=1 error = %v, want nil", err)
	}

	// Frame exactly equal to configured maxFrameSize should be accepted.
	totalLen := HeaderLength + len(payload)
	rExact := NewReader(bytes.NewReader(pkt), WithMaxFrameSize(totalLen))
	if _, err := rExact.ReadFrame(); err != nil {
		t.Fatalf("ReadFrame() with maxFrameSize=totalLen error = %v, want nil", err)
	}
}

func TestReaderHeaderValidationErrors(t *testing.T) {
	basePayload := []byte{0x01, 0x02, 0x03}
	validPkt, err := Encode(basePayload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

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
			name: "invalid reserved",
			mutate: func(b []byte) []byte {
				cp := append([]byte(nil), b...)
				cp[1] = 0x01
				return cp
			},
			wantErr: ErrInvalidReserved,
		},
		{
			name: "invalid length below minimum",
			mutate: func(b []byte) []byte {
				cp := append([]byte(nil), b...)
				cp[2] = 0x00
				cp[3] = 0x06
				return cp
			},
			wantErr: ErrInvalidLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.mutate(validPkt)
			r := NewReader(bytes.NewReader(input))
			_, err := r.ReadFrame()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ReadFrame() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

type byteChunkReader struct {
	src []byte
	pos int
}

func (b *byteChunkReader) Read(p []byte) (int, error) {
	if b.pos >= len(b.src) {
		return 0, io.EOF
	}
	p[0] = b.src[b.pos]
	b.pos++
	return 1, nil
}

func TestReaderChunkedUnderlyingReader(t *testing.T) {
	payload := []byte("chunked")
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	r := NewReader(&byteChunkReader{src: pkt})

	got, err := r.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload mismatch: got %v want %v", got, payload)
	}
}

func TestReaderReadFrameAllocatesIndependentPayload(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	r := NewReader(bytes.NewReader(pkt))
	got, err := r.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	// Mutate the returned payload and verify the original packet buffer is unchanged.
	orig := append([]byte(nil), pkt...)
	got[0] ^= 0xff
	if !bytes.Equal(pkt, orig) {
		t.Fatalf("expected original packet buffer to remain unchanged")
	}
}
