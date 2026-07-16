// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"testing"
)

func FuzzDecodePacket(f *testing.F) {
	seeds := [][]byte{
		{0x03, 0x00, 0x00, 0x07, 0x01, 0x02, 0x03},
		{0x03, 0xff, 0x00, 0x07, 0x01, 0x02, 0x03},
		{0x02, 0x00, 0x00, 0x07, 0x01, 0x02, 0x03},
		{0x03, 0x00, 0x00, 0x07, 0x01, 0x02, 0x03, 0x00},
		{0x03, 0x00, 0x00},
		{0x03, 0x00, 0x00, 0x06, 0x01, 0x02},
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		payload, err := DecodePacket(data)
		if err != nil {
			return
		}
		if len(data) < MinPacketLength {
			t.Fatalf("success with len=%d < MinPacketLength", len(data))
		}
		if data[0] != Version {
			t.Fatal("success with invalid version")
		}
		if len(data) != HeaderLength+len(payload) {
			t.Fatalf("trailing or length inconsistency: packet=%d payload=%d", len(data), len(payload))
		}
		if len(payload) < MinPayloadLength {
			t.Fatalf("payload too short: %d", len(payload))
		}
	})
}

func FuzzEncodeDecodePacket(f *testing.F) {
	f.Add([]byte{0x01, 0x02, 0x03})
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, payload []byte) {
		if len(payload) < MinPayloadLength || len(payload) > MaxPayloadLength {
			return
		}
		pkt, err := EncodePacket(payload)
		if err != nil {
			t.Fatalf("EncodePacket: %v", err)
		}
		got, err := DecodePacket(pkt)
		if err != nil {
			t.Fatalf("DecodePacket: %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("round-trip mismatch")
		}
		if pkt[1] != 0 {
			t.Fatal("reserved not zero")
		}
	})
}

func FuzzReaderChunking(f *testing.F) {
	f.Add([]byte{0x01, 0x02, 0x03}, 1)
	f.Add([]byte{0x0a, 0x0b, 0x0c, 0x0d}, 3)
	f.Fuzz(func(t *testing.T, payload []byte, chunkSize int) {
		if len(payload) < MinPayloadLength || len(payload) > 256 {
			return
		}
		if chunkSize <= 0 {
			chunkSize = 1
		}
		if chunkSize > 64 {
			chunkSize = 64
		}
		pkt, err := EncodePacket(payload)
		if err != nil {
			t.Fatal(err)
		}
		chunks := make([]int, 0, len(pkt))
		for remain := len(pkt); remain > 0; remain -= chunkSize {
			n := chunkSize
			if n > remain {
				n = remain
			}
			chunks = append(chunks, n)
		}
		cr := &chunkReader{data: pkt, chunks: chunks}
		r, err := NewReader(cr, ReaderConfig{})
		if err != nil {
			t.Fatal(err)
		}
		got, err := r.ReadPacket()
		if err != nil {
			t.Fatalf("ReadPacket: %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("mismatch")
		}
	})
}

func FuzzReservedPolicy(f *testing.F) {
	f.Add(byte(0))
	f.Add(byte(1))
	f.Add(byte(0xff))
	f.Fuzz(func(t *testing.T, reserved byte) {
		pkt := []byte{Version, reserved, 0x00, 0x07, 'x', 'y', 'z'}
		got, err := DecodePacket(pkt)
		if err != nil {
			t.Fatalf("reserved=%#x: %v", reserved, err)
		}
		if !bytes.Equal(got, []byte{'x', 'y', 'z'}) {
			t.Fatalf("payload mismatch")
		}
	})
}
