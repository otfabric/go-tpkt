// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"io"
	"testing"
)

// allocSizes are the payload sizes of the allocation budgets.
var allocSizes = []int{MinPayloadLength, 64, 256, 1024, 8192, 8193, MaxPayloadLength}

// TestReadPacketAllocations: ReadPacket allocates the payload it returns and
// nothing else.
func TestReadPacketAllocations(t *testing.T) {
	for _, n := range allocSizes {
		pkt, err := EncodePacket(bytes.Repeat([]byte{0xA5}, n))
		if err != nil {
			t.Fatal(err)
		}
		src := bytes.NewReader(pkt)
		r, err := NewReader(src, ReaderConfig{})
		if err != nil {
			t.Fatal(err)
		}
		allocs := testing.AllocsPerRun(100, func() {
			src.Reset(pkt)
			if _, err := r.ReadPacket(); err != nil {
				t.Fatal(err)
			}
		})
		if allocs != 1 {
			t.Errorf("payload %d: %v allocations per ReadPacket, want 1", n, allocs)
		}
	}
}

// TestWritePacketAllocations: after the first packet, WritePacket does not
// allocate for a packet of at most writerKeepLength octets and allocates
// once for a larger one.
func TestWritePacketAllocations(t *testing.T) {
	for _, n := range allocSizes {
		w, err := NewWriter(io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		payload := bytes.Repeat([]byte{0x5A}, n)
		if err := w.WritePacket(payload); err != nil {
			t.Fatal(err)
		}
		allocs := testing.AllocsPerRun(100, func() {
			if err := w.WritePacket(payload); err != nil {
				t.Fatal(err)
			}
		})
		want := 0.0
		if HeaderLength+n > writerKeepLength {
			want = 1
		}
		if allocs != want {
			t.Errorf("payload %d: %v allocations per WritePacket, want %v", n, allocs, want)
		}
	}
}

// TestWriterKeepsBoundedBuffer: a Writer never keeps more than
// writerKeepLength octets, and a large packet does not replace the kept
// buffer.
func TestWriterKeepsBoundedBuffer(t *testing.T) {
	w, err := NewWriter(io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	steps := []struct {
		payload int
		wantCap int
	}{
		{MaxPayloadLength, 0},
		{64, HeaderLength + 64},
		{8192, writerKeepLength},
		{MaxPayloadLength, writerKeepLength},
		{MinPayloadLength, writerKeepLength},
	}
	for i, s := range steps {
		if err := w.WritePacket(make([]byte, s.payload)); err != nil {
			t.Fatal(err)
		}
		kept := 0
		if w.buf != nil {
			kept = cap(*w.buf)
		}
		if kept != s.wantCap {
			t.Fatalf("step %d (payload %d): kept %d octets, want %d", i, s.payload, kept, s.wantCap)
		}
	}
}

// retainingWriter keeps the slices it is given, which io.Writer forbids.
type retainingWriter struct{ kept [][]byte }

func (r *retainingWriter) Write(p []byte) (int, error) {
	r.kept = append(r.kept, p)
	return len(p), nil
}

// TestWriterReusesBuffer documents the io.Writer requirement: each packet is
// written correctly, and the underlying writer sees the same buffer again.
func TestWriterReusesBuffer(t *testing.T) {
	var out bytes.Buffer
	rw := &retainingWriter{}
	w, err := NewWriter(io.MultiWriter(&out, rw))
	if err != nil {
		t.Fatal(err)
	}
	first, second := []byte{1, 2, 3, 4}, []byte{9, 8, 7, 6}
	for _, p := range [][]byte{first, second} {
		if err := w.WritePacket(p); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := EncodePacket(first)
	b, _ := EncodePacket(second)
	if want := append(a, b...); !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("stream % x, want % x", out.Bytes(), want)
	}
	if len(rw.kept) != 2 || &rw.kept[0][0] != &rw.kept[1][0] {
		t.Fatalf("Writer did not reuse its buffer")
	}
	if !bytes.Equal(rw.kept[0], b) {
		t.Fatalf("kept first slice % x, want it overwritten by the second packet % x", rw.kept[0], b)
	}
}

// TestWriterComparable: Writer and Reader stay comparable, as in v1.2.1
// (gorelease treats losing that as an incompatible change).
func TestWriterComparable(t *testing.T) {
	w1, _ := NewWriter(io.Discard)
	w2, _ := NewWriter(io.Discard)
	if *w1 != *w2 {
		t.Fatal("fresh Writers over the same stream differ")
	}
	r1, _ := NewReader(bytes.NewReader(nil), ReaderConfig{})
	if *r1 != *r1 {
		t.Fatal("Reader not equal to itself")
	}
}

// TestEncodeIntoRefusesOutOfRange: encodeInto is only called with a checked
// length; a wrong one panics instead of writing a truncated length.
func TestEncodeIntoRefusesOutOfRange(t *testing.T) {
	for _, n := range []int{MinPacketLength - 1, MaxPacketLength + 1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("packet length %d: no panic", n)
				}
			}()
			encodeInto(make([]byte, n), make([]byte, max(0, n-HeaderLength)))
		}()
	}
}
