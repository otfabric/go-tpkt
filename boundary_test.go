// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"testing"
)

// Boundary tables for the length field and the receive limit, and the
// allocation budget of the receive limit. The cases are explicit, so that a
// change of one bound by one octet fails a named case.

// patterned returns n octets that differ from one position to the next.
func patterned(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

// headerDeclaring returns a version-3 TPKT header that declares total octets.
func headerDeclaring(total int) []byte {
	return []byte{Version, 0x00, byte(total >> 8), byte(total)}
}

// rawPacket returns a header that declares total octets, followed by the
// octets the declaration covers. A declaration below the minimum still gets
// a minimum payload, so the input is never shorter than a legal packet.
func rawPacket(total int) []byte {
	pkt := headerDeclaring(total)
	return append(pkt, patterned(max(total, MinPacketLength)-HeaderLength)...)
}

// TestDeclaredLengthBoundaries gives the same input to DecodePacket and to
// Reader.ReadPacket for each boundary of the declared length. The length
// field has 16 bits, so 65535 is both the protocol maximum and the largest
// value a header can declare.
func TestDeclaredLengthBoundaries(t *testing.T) {
	cases := []struct {
		declared int
		wantErr  error
	}{
		{declared: 0, wantErr: ErrInvalidPacketLength},
		{declared: 1, wantErr: ErrInvalidPacketLength},
		{declared: HeaderLength, wantErr: ErrInvalidPacketLength},
		{declared: MinPacketLength - 1, wantErr: ErrInvalidPacketLength},
		{declared: MinPacketLength},
		{declared: MinPacketLength + 1},
		{declared: MaxPacketLength - 1},
		{declared: MaxPacketLength},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("declared=%d", tc.declared), func(t *testing.T) {
			pkt := rawPacket(tc.declared)
			check := func(entry string, payload []byte, err error) {
				t.Helper()
				if tc.wantErr != nil {
					if !errors.Is(err, tc.wantErr) {
						t.Fatalf("%s: got %v, want %v", entry, err, tc.wantErr)
					}
					if payload != nil {
						t.Fatalf("%s: payload of %d octets with an error", entry, len(payload))
					}
					return
				}
				if err != nil {
					t.Fatalf("%s: %v", entry, err)
				}
				if !bytes.Equal(payload, pkt[HeaderLength:]) {
					t.Fatalf("%s: payload of %d octets differs from the %d octets sent",
						entry, len(payload), len(pkt)-HeaderLength)
				}
			}

			payload, err := DecodePacket(pkt)
			check("DecodePacket", payload, err)

			r, err := NewReader(bytes.NewReader(pkt), ReaderConfig{})
			if err != nil {
				t.Fatal(err)
			}
			payload, err = r.ReadPacket()
			check("ReadPacket", payload, err)
		})
	}
}

// TestRoundTripPayloadBoundaries sends the smallest and the largest payload,
// and their neighbours, through both pairs of entry points.
func TestRoundTripPayloadBoundaries(t *testing.T) {
	for _, n := range []int{MinPayloadLength, MinPayloadLength + 1, MaxPayloadLength - 1, MaxPayloadLength} {
		t.Run(fmt.Sprintf("payload=%d", n), func(t *testing.T) {
			payload := patterned(n)

			pkt, err := EncodePacket(payload)
			if err != nil {
				t.Fatalf("EncodePacket: %v", err)
			}
			if got := int(binary.BigEndian.Uint16(pkt[2:4])); got != HeaderLength+n || len(pkt) != HeaderLength+n {
				t.Fatalf("EncodePacket: declared %d, %d octets, want %d for both", got, len(pkt), HeaderLength+n)
			}
			decoded, err := DecodePacket(pkt)
			if err != nil {
				t.Fatalf("DecodePacket: %v", err)
			}
			if !bytes.Equal(decoded, payload) {
				t.Fatal("DecodePacket: payload differs from the one encoded")
			}

			var stream bytes.Buffer
			w, err := NewWriter(&stream)
			if err != nil {
				t.Fatal(err)
			}
			if err := w.WritePacket(payload); err != nil {
				t.Fatalf("WritePacket: %v", err)
			}
			if !bytes.Equal(stream.Bytes(), pkt) {
				t.Fatal("WritePacket: octets on the stream differ from EncodePacket")
			}
			r, err := NewReader(&stream, ReaderConfig{})
			if err != nil {
				t.Fatal(err)
			}
			read, err := r.ReadPacket()
			if err != nil {
				t.Fatalf("ReadPacket: %v", err)
			}
			if !bytes.Equal(read, payload) {
				t.Fatal("ReadPacket: payload differs from the one written")
			}
			if _, err := r.ReadPacket(); err != io.EOF {
				t.Fatalf("ReadPacket behind the packet: got %v, want io.EOF", err)
			}
		})
	}
}

// TestNewReaderMaxPacketLengthBoundaries covers ReaderConfig.MaxPacketLength
// at and around both ends of its range, at zero and at the limits of int.
func TestNewReaderMaxPacketLengthBoundaries(t *testing.T) {
	cases := []struct {
		value int
		want  int // the limit in effect; 0 when the value is refused
	}{
		{value: math.MinInt},
		{value: math.MinInt32},
		{value: -1},
		{value: 0, want: MaxPacketLength},
		{value: 1},
		{value: MinPacketLength - 1},
		{value: MinPacketLength, want: MinPacketLength},
		{value: MinPacketLength + 1, want: MinPacketLength + 1},
		{value: MaxPacketLength - 1, want: MaxPacketLength - 1},
		{value: MaxPacketLength, want: MaxPacketLength},
		{value: MaxPacketLength + 1},
		{value: math.MaxInt32},
		{value: math.MaxInt},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("value=%d", tc.value), func(t *testing.T) {
			r, err := NewReader(bytes.NewReader(nil), ReaderConfig{MaxPacketLength: tc.value})
			if tc.want == 0 {
				if !errors.Is(err, ErrInvalidMaxPacketLength) {
					t.Fatalf("got %v, want ErrInvalidMaxPacketLength", err)
				}
				if r != nil {
					t.Fatal("a Reader was returned with an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if r.maxPacketLength != tc.want {
				t.Fatalf("limit in effect = %d, want %d", r.maxPacketLength, tc.want)
			}
		})
	}
}

// TestReaderLimitBoundaries reads a packet at the receive limit and one
// octet above it, for small, mid-size and full-size limits. A packet at the
// limit is delivered. A packet above it is refused after its header, and the
// rest of it stays on the stream.
func TestReaderLimitBoundaries(t *testing.T) {
	cases := []struct {
		limit   int // 0 selects the default, MaxPacketLength
		total   int
		wantErr error
	}{
		{limit: MinPacketLength, total: MinPacketLength},
		{limit: MinPacketLength, total: MinPacketLength + 1, wantErr: ErrPacketTooLarge},
		{limit: 12, total: 11},
		{limit: 12, total: 12},
		{limit: 12, total: 13, wantErr: ErrPacketTooLarge},
		{limit: 4096, total: 4096},
		{limit: 4096, total: 4097, wantErr: ErrPacketTooLarge},
		{limit: MaxPacketLength - 1, total: MaxPacketLength - 1},
		{limit: MaxPacketLength - 1, total: MaxPacketLength, wantErr: ErrPacketTooLarge},
		{limit: MaxPacketLength, total: MaxPacketLength},
		{limit: 0, total: MaxPacketLength},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("limit=%d/packet=%d", tc.limit, tc.total), func(t *testing.T) {
			payload := patterned(tc.total - HeaderLength)
			pkt, err := EncodePacket(payload)
			if err != nil {
				t.Fatal(err)
			}
			src := bytes.NewReader(pkt)
			r, err := NewReader(src, ReaderConfig{MaxPacketLength: tc.limit})
			if err != nil {
				t.Fatal(err)
			}
			got, err := r.ReadPacket()
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got %v, want %v", err, tc.wantErr)
				}
				if got != nil {
					t.Fatalf("payload of %d octets with an error", len(got))
				}
				if src.Len() != len(payload) {
					t.Fatalf("%d octets left on the stream, want the %d of the payload", src.Len(), len(payload))
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadPacket: %v", err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatal("payload differs from the one sent")
			}
			if _, err := r.ReadPacket(); err != io.EOF {
				t.Fatalf("ReadPacket behind the packet: got %v, want io.EOF", err)
			}
		})
	}
}

// allocatedPerCall returns the bytes allocated by one call of fn, averaged
// over runs calls.
func allocatedPerCall(runs int, fn func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for range runs {
		fn()
	}
	runtime.ReadMemStats(&after)
	return (after.TotalAlloc - before.TotalAlloc) / uint64(runs)
}

// allocationSlack is what one NewReader and one failing ReadPacket may
// allocate besides a payload: the Reader, the error it returns and the one
// it records. An allocation of a declared size in the cases below is at
// least twice as large.
const allocationSlack = 2 << 10

// TestReaderOversizeAllocationBudget: a header that declares more than the
// receive limit is refused before anything of the declared size is
// allocated. The stream holds the header only, so a Reader that went on to
// the payload would return another error. The budget is in bytes per call.
// Each call has a Reader of its own: a Reader that refused a header stays
// failed and does not read again.
func TestReaderOversizeAllocationBudget(t *testing.T) {
	cases := []struct {
		limit    int
		declared int
	}{
		{limit: MinPacketLength, declared: MaxPacketLength},
		{limit: 12, declared: MaxPacketLength},
		{limit: 4096, declared: 4097},
		{limit: 4096, declared: MaxPacketLength},
		{limit: MaxPacketLength - 1, declared: MaxPacketLength},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("limit=%d/declared=%d", tc.limit, tc.declared), func(t *testing.T) {
			hdr := headerDeclaring(tc.declared)
			src := bytes.NewReader(hdr)
			got := allocatedPerCall(100, func() {
				src.Reset(hdr)
				r, err := NewReader(src, ReaderConfig{MaxPacketLength: tc.limit})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := r.ReadPacket(); !errors.Is(err, ErrPacketTooLarge) {
					t.Fatalf("got %v, want ErrPacketTooLarge", err)
				}
			})
			if got > allocationSlack {
				t.Fatalf("allocated %d bytes per refused header that declares %d octets, budget %d",
					got, tc.declared, allocationSlack)
			}
		})
	}
}

// TestReaderAllocationBoundedByLimit: a header at the receive limit, with no
// payload behind it, makes ReadPacket allocate the payload it declares and
// nothing that grows beyond it. This is the most a peer can make one
// ReadPacket call hold: the limit minus the header.
func TestReaderAllocationBoundedByLimit(t *testing.T) {
	for _, limit := range []int{MinPacketLength, 4096, MaxPacketLength} {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			hdr := headerDeclaring(limit)
			src := bytes.NewReader(hdr)
			got := allocatedPerCall(100, func() {
				src.Reset(hdr)
				r, err := NewReader(src, ReaderConfig{MaxPacketLength: limit})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := r.ReadPacket(); !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Fatalf("got %v, want io.ErrUnexpectedEOF", err)
				}
			})
			if budget := uint64(limit - HeaderLength + allocationSlack); got > budget {
				t.Fatalf("allocated %d bytes per call at limit %d, budget %d", got, limit, budget)
			}
		})
	}
}
