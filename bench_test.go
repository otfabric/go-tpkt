// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"fmt"
	"testing"
)

// The benchmarks run over three payload sizes: the minimum, a mid-size TPDU
// and the maximum, which makes a full-size packet of 65535 octets. Each
// reports allocations; those that copy the payload also report, with
// SetBytes, the throughput in octets on the wire (header included). b.Loop
// keeps the results alive; each benchmark also checks the result of its
// last call, so a call that fails or returns less is not measured as a fast
// one.
var benchPayloadSizes = []int{MinPayloadLength, 1024, MaxPayloadLength}

// BenchmarkEncodePacket measures one EncodePacket: the length check, the
// allocation of the packet and the copy of the payload into it.
func BenchmarkEncodePacket(b *testing.B) {
	for _, n := range benchPayloadSizes {
		b.Run(fmt.Sprintf("payload=%d", n), func(b *testing.B) {
			payload := patterned(n)
			var pkt []byte
			b.ReportAllocs()
			b.SetBytes(int64(HeaderLength + n))
			for b.Loop() {
				var err error
				pkt, err = EncodePacket(payload)
				if err != nil {
					b.Fatal(err)
				}
			}
			if len(pkt) != HeaderLength+n {
				b.Fatalf("packet of %d octets, want %d", len(pkt), HeaderLength+n)
			}
		})
	}
}

// BenchmarkDecodePacket measures one DecodePacket of a packet in memory:
// the header and length checks. The result aliases the input, so nothing is
// allocated or copied and the cost does not grow with the payload; no
// throughput is reported for that reason.
func BenchmarkDecodePacket(b *testing.B) {
	for _, n := range benchPayloadSizes {
		b.Run(fmt.Sprintf("payload=%d", n), func(b *testing.B) {
			pkt, err := EncodePacket(patterned(n))
			if err != nil {
				b.Fatal(err)
			}
			var payload []byte
			b.ReportAllocs()
			for b.Loop() {
				payload, err = DecodePacket(pkt)
				if err != nil {
					b.Fatal(err)
				}
			}
			if len(payload) != n {
				b.Fatalf("payload of %d octets, want %d", len(payload), n)
			}
		})
	}
}

// BenchmarkReadPacket measures one Reader.ReadPacket of a packet held in
// memory: the header read and its checks, the allocation of the payload and
// the copy out of the stream. The Reader is built once, outside the loop;
// the loop rewinds the in-memory stream, which allocates nothing. No network
// is involved.
func BenchmarkReadPacket(b *testing.B) {
	for _, n := range benchPayloadSizes {
		b.Run(fmt.Sprintf("payload=%d", n), func(b *testing.B) {
			pkt, err := EncodePacket(patterned(n))
			if err != nil {
				b.Fatal(err)
			}
			src := bytes.NewReader(pkt)
			r, err := NewReader(src, ReaderConfig{})
			if err != nil {
				b.Fatal(err)
			}
			var payload []byte
			b.ReportAllocs()
			b.SetBytes(int64(len(pkt)))
			for b.Loop() {
				src.Reset(pkt)
				payload, err = r.ReadPacket()
				if err != nil {
					b.Fatal(err)
				}
			}
			if len(payload) != n {
				b.Fatalf("payload of %d octets, want %d", len(payload), n)
			}
		})
	}
}

// lastWrite is an io.Writer that accepts everything and keeps the size of
// the last Write.
type lastWrite struct {
	n int
}

func (l *lastWrite) Write(p []byte) (int, error) {
	l.n = len(p)
	return len(p), nil
}

// BenchmarkWritePacket measures one Writer.WritePacket to a writer that
// accepts every Write in full and stores nothing: the encoding of the
// packet and one Write call. The Writer is built once, outside the loop. No
// network is involved.
func BenchmarkWritePacket(b *testing.B) {
	for _, n := range benchPayloadSizes {
		b.Run(fmt.Sprintf("payload=%d", n), func(b *testing.B) {
			payload := patterned(n)
			dst := &lastWrite{}
			w, err := NewWriter(dst)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.SetBytes(int64(HeaderLength + n))
			for b.Loop() {
				if err := w.WritePacket(payload); err != nil {
					b.Fatal(err)
				}
			}
			if dst.n != HeaderLength+n {
				b.Fatalf("last write of %d octets, want %d", dst.n, HeaderLength+n)
			}
		})
	}
}
