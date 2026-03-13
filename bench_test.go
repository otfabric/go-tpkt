package tpkt

import (
	"bytes"
	"testing"
)

func BenchmarkEncodeSmall(b *testing.B) {
	payload := []byte{0x01, 0x02, 0x03}
	for i := 0; i < b.N; i++ {
		if _, err := Encode(payload); err != nil {
			b.Fatalf("Encode error: %v", err)
		}
	}
}

func BenchmarkDecodeSmall(b *testing.B) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := Encode(payload)
	if err != nil {
		b.Fatalf("Encode error: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Decode(pkt); err != nil {
			b.Fatalf("Decode error: %v", err)
		}
	}
}

func BenchmarkReaderSmall(b *testing.B) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := Encode(payload)
	if err != nil {
		b.Fatalf("Encode error: %v", err)
	}

	for i := 0; i < b.N; i++ {
		r := NewReader(bytes.NewReader(pkt))
		if _, err := r.ReadFrame(); err != nil {
			b.Fatalf("ReadFrame error: %v", err)
		}
	}
}

func BenchmarkWriterSmall(b *testing.B) {
	payload := []byte{0x01, 0x02, 0x03}
	var buf bytes.Buffer

	for i := 0; i < b.N; i++ {
		buf.Reset()
		w := NewWriter(&buf)
		if _, err := w.WriteFrame(payload); err != nil {
			b.Fatalf("WriteFrame error: %v", err)
		}
	}
}
