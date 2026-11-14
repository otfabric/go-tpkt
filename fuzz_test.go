// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
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
			n := min(chunkSize, remain)
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

// errStreamFault is the error a steadyReader returns for its fault.
var errStreamFault = errors.New("stream fault")

// steadyReader hands out data in pieces of at most chunk octets (as much as
// is asked for when chunk is 0). When faultAt is not negative, the Read that
// would deliver the octet at that offset returns errStreamFault instead,
// once, and the stream goes on behind it, as a connection does behind an
// expired deadline. It counts the octets it delivered and the Read calls.
type steadyReader struct {
	data    []byte
	pos     int
	chunk   int
	faultAt int
	faulted bool
	calls   int
}

func (s *steadyReader) Read(p []byte) (int, error) {
	s.calls++
	if s.pos == s.faultAt && !s.faulted {
		s.faulted = true
		return 0, errStreamFault
	}
	if s.pos >= len(s.data) {
		return 0, io.EOF
	}
	n := min(len(p), len(s.data)-s.pos)
	if s.chunk > 0 {
		n = min(n, s.chunk)
	}
	if !s.faulted && s.faultAt > s.pos {
		n = min(n, s.faultAt-s.pos)
	}
	copy(p, s.data[s.pos:s.pos+n])
	s.pos += n
	return n, nil
}

// walkStep says what one ReadPacket does with the octets in rest on a stream
// that does not fail: how many octets it consumes, the error it returns (nil
// for a packet), and whether it runs into the end of the stream.
func walkStep(rest []byte, maxLen int) (span int, want error, hitsEnd bool) {
	switch {
	case len(rest) == 0:
		return 0, io.EOF, true
	case len(rest) < HeaderLength:
		return len(rest), io.ErrUnexpectedEOF, true
	}
	declared := int(binary.BigEndian.Uint16(rest[2:4]))
	switch {
	case rest[0] != Version:
		return HeaderLength, ErrInvalidVersion, false
	case declared < MinPacketLength:
		return HeaderLength, ErrInvalidPacketLength, false
	case declared > maxLen:
		return HeaderLength, ErrPacketTooLarge, false
	case declared > len(rest):
		return len(rest), io.ErrUnexpectedEOF, true
	}
	return declared, nil, false
}

// FuzzReaderStream gives arbitrary octets to Reader.ReadPacket, with an
// arbitrary receive limit and chunking and one stream error at an arbitrary
// offset, and compares every call with a walk over the same octets:
//
//   - a payload is returned only for a header with version 3 and a declared
//     length from MinPacketLength up to the limit whose octets are all on
//     the stream; it equals those octets, and the Reader consumed that
//     packet and nothing behind it;
//   - io.EOF itself is returned only on a packet boundary; a stream that
//     ends inside a packet is io.ErrUnexpectedEOF and never matches io.EOF;
//   - a refused header returns the sentinel of its reason, and the Reader
//     consumed the header only;
//   - the stream error is returned wrapped, and never as an end of stream;
//   - an error on a packet boundary (the stream error, io.EOF) leaves the
//     Reader usable: the walk goes on at the same offset;
//   - after any error inside a packet (a refused header, the end of the
//     stream, the stream error) every later call returns an error that
//     matches ErrStreamMisaligned and the first error, and reads nothing;
//   - no error comes with a payload.
func FuzzReaderStream(f *testing.F) {
	seeds := []struct {
		data  []byte
		limit uint16
		chunk uint8
		fault uint16 // 0: no stream error; otherwise one before the octet at offset fault-1
	}{
		{data: []byte{3, 0, 0, 7, 1, 2, 3, 3, 9, 0, 8, 1, 2, 3, 4}, limit: 0, chunk: 1},
		{data: []byte{3, 0, 0, 8, 1, 2, 3, 4}, limit: 8, chunk: 0},
		{data: []byte{3, 0, 0, 8, 1, 2, 3, 4}, limit: 7, chunk: 0},
		{data: []byte{3, 0, 0xff, 0xff, 1}, limit: 7, chunk: 3},
		{data: []byte{3, 0, 0xff, 0xff, 1}, limit: 0, chunk: 3},
		{data: []byte{3, 0, 0, 6, 1, 2}, limit: 0, chunk: 4},
		{data: []byte{2, 0, 0, 7, 1, 2, 3}, limit: 100, chunk: 200},
		{data: []byte{3, 0, 0, 7, 1, 2, 3}, limit: 3, chunk: 0},
		{data: []byte{3, 0, 0, 7, 1, 2, 3, 3, 0}, limit: 0, chunk: 2},
		{data: []byte{3, 0, 0, 9, 1}, limit: 0, chunk: 0},
		{data: nil, limit: 0, chunk: 0},
		// The stream error before the first packet, between two packets and
		// behind the last one: the Reader stays usable.
		{data: []byte{3, 0, 0, 7, 1, 2, 3, 3, 0, 0, 7, 4, 5, 6}, limit: 0, chunk: 0, fault: 1},
		{data: []byte{3, 0, 0, 7, 1, 2, 3, 3, 0, 0, 7, 4, 5, 6}, limit: 0, chunk: 0, fault: 8},
		{data: []byte{3, 0, 0, 7, 1, 2, 3, 3, 0, 0, 7, 4, 5, 6}, limit: 0, chunk: 5, fault: 15},
		// The stream error inside a header, behind a header and inside a
		// payload: the Reader stays failed.
		{data: []byte{3, 0, 0, 7, 1, 2, 3, 3, 0, 0, 7, 4, 5, 6}, limit: 0, chunk: 0, fault: 3},
		{data: []byte{3, 0, 0, 7, 1, 2, 3, 3, 0, 0, 7, 4, 5, 6}, limit: 0, chunk: 0, fault: 5},
		{data: []byte{3, 0, 0, 7, 3, 0, 0, 7, 3, 0, 0, 7, 1, 2}, limit: 0, chunk: 1, fault: 13},
		// The stream error where a truncated packet ends.
		{data: []byte{3, 0, 0, 9, 1}, limit: 0, chunk: 0, fault: 6},
	}
	for _, s := range seeds {
		f.Add(s.data, s.limit, s.chunk, s.fault)
	}

	f.Fuzz(func(t *testing.T, data []byte, limit uint16, chunk uint8, fault uint16) {
		maxLen := int(limit)
		src := &steadyReader{data: data, chunk: int(chunk), faultAt: int(fault) - 1}
		r, err := NewReader(src, ReaderConfig{MaxPacketLength: maxLen})
		if maxLen != 0 && maxLen < MinPacketLength {
			if !errors.Is(err, ErrInvalidMaxPacketLength) || r != nil {
				t.Fatalf("NewReader(limit=%d): got %v, want ErrInvalidMaxPacketLength and no Reader", maxLen, err)
			}
			return
		}
		if err != nil {
			t.Fatalf("NewReader(limit=%d): %v", maxLen, err)
		}
		if maxLen == 0 {
			maxLen = MaxPacketLength
		}

		// staysFailed checks the calls behind an error inside a packet.
		staysFailed := func(off int, cause error) {
			t.Helper()
			pos, calls := src.pos, src.calls
			for range 2 {
				payload, err := r.ReadPacket()
				if payload != nil || !errors.Is(err, ErrStreamMisaligned) || !errors.Is(err, cause) {
					t.Fatalf("offset %d: behind %v: got %d octets and %v, want no payload and ErrStreamMisaligned with the cause",
						off, cause, len(payload), err)
				}
			}
			if src.pos != pos || src.calls != calls {
				t.Fatalf("offset %d: behind %v: %d reads and %d octets taken from the stream, want none",
					off, cause, src.calls-calls, src.pos-pos)
			}
		}

		for off := 0; ; {
			faultPending := src.faultAt >= 0 && !src.faulted
			payload, err := r.ReadPacket()
			if err != nil && payload != nil {
				t.Fatalf("offset %d: payload of %d octets with error %v", off, len(payload), err)
			}
			rest := data[off:]
			span, want, hitsEnd := walkStep(rest, maxLen)

			// The stream error falls into this call when the call still asks
			// for the octet at faultAt.
			if at := src.faultAt; faultPending && at >= off && (at < off+span || hitsEnd && at == off+span) {
				if !errors.Is(err, errStreamFault) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					t.Fatalf("offset %d: stream error at %d: got %v, want the stream error and no end of stream", off, at, err)
				}
				if src.pos != at {
					t.Fatalf("offset %d: stream error at %d: stream position %d", off, at, src.pos)
				}
				if at > off {
					staysFailed(off, errStreamFault)
					return
				}
				// On a packet boundary: the Reader stays usable.
				if errors.Is(err, ErrStreamMisaligned) {
					t.Fatalf("offset %d: got %v for a stream error on a packet boundary", off, err)
				}
				continue
			}

			switch {
			case want == nil:
				// A complete packet within the limit.
				if err != nil {
					t.Fatalf("offset %d: packet of %d octets within limit %d refused: %v", off, span, maxLen, err)
				}
				if !bytes.Equal(payload, rest[HeaderLength:span]) {
					t.Fatalf("offset %d: payload differs from the octets on the stream", off)
				}
				if HeaderLength+len(payload) > maxLen {
					t.Fatalf("offset %d: packet of %d octets above limit %d", off, HeaderLength+len(payload), maxLen)
				}
				decoded, err := DecodePacket(rest[:span])
				if err != nil || !bytes.Equal(decoded, payload) {
					t.Fatalf("offset %d: DecodePacket disagrees with ReadPacket: %v", off, err)
				}
				off += span
				if src.pos != off {
					t.Fatalf("stream position %d behind a packet that ends at %d", src.pos, off)
				}

			case span == 0:
				// The end of the stream on a packet boundary, also when asked again.
				if err != io.EOF {
					t.Fatalf("offset %d: got %v on a packet boundary, want io.EOF", off, err)
				}
				if _, err := r.ReadPacket(); err != io.EOF {
					t.Fatalf("offset %d: got %v for a second call at the end of the stream, want io.EOF", off, err)
				}
				return

			default:
				// A refused header, or the end of the stream inside a packet.
				if !errors.Is(err, want) || errors.Is(err, io.EOF) {
					t.Fatalf("offset %d: header % x, %d octets, limit %d: got %v, want %v",
						off, rest[:min(len(rest), HeaderLength)], len(rest), maxLen, err, want)
				}
				if src.pos != off+span {
					t.Fatalf("offset %d: %d octets consumed for %v, want %d", off, src.pos-off, want, span)
				}
				staysFailed(off, want)
				return
			}
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
