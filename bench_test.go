// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"testing"
)

func BenchmarkEncodePacketSmall(b *testing.B) {
	payload := []byte{0x01, 0x02, 0x03}
	for i := 0; i < b.N; i++ {
		if _, err := EncodePacket(payload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodePacketSmall(b *testing.B) {
	pkt, err := EncodePacket([]byte{0x01, 0x02, 0x03})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DecodePacket(pkt); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadWritePacket(b *testing.B) {
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	var buf bytes.Buffer
	w, err := NewWriter(&buf)
	if err != nil {
		b.Fatal(err)
	}
	if err := w.WritePacket(payload); err != nil {
		b.Fatal(err)
	}
	data := buf.Bytes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := NewReader(bytes.NewReader(data), ReaderConfig{})
		if err != nil {
			b.Fatal(err)
		}
		if _, err := r.ReadPacket(); err != nil {
			b.Fatal(err)
		}
	}
}
