// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "minimum payload (3 bytes)", payload: []byte{0x01, 0x02, 0x03}},
		{name: "small payload", payload: []byte{0xde, 0xad, 0xbe, 0xef}},
		{name: "arbitrary payload", payload: []byte("hello, tpkt")},
		{name: "non-multiple-of-4 total length", payload: []byte{0x01, 0x02, 0x03, 0x04, 0x05}}, // total 9
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt, err := EncodePacket(tt.payload)
			if err != nil {
				t.Fatalf("EncodePacket() error = %v", err)
			}
			if pkt[0] != Version || pkt[1] != 0 {
				t.Fatalf("header version/reserved = %d/%d, want 3/0", pkt[0], pkt[1])
			}
			got, err := DecodePacket(pkt)
			if err != nil {
				t.Fatalf("DecodePacket() error = %v", err)
			}
			if !bytes.Equal(got, tt.payload) {
				t.Fatalf("payload mismatch: got %v want %v", got, tt.payload)
			}
		})
	}
}

func TestEncodePacketLengthValidation(t *testing.T) {
	t.Run("empty payload", func(t *testing.T) {
		if _, err := EncodePacket(nil); !errors.Is(err, ErrPayloadTooShort) {
			t.Fatalf("got %v, want ErrPayloadTooShort", err)
		}
	})
	t.Run("payload shorter than 3", func(t *testing.T) {
		if _, err := EncodePacket([]byte{0x01, 0x02}); !errors.Is(err, ErrPayloadTooShort) {
			t.Fatalf("got %v, want ErrPayloadTooShort", err)
		}
	})
	t.Run("max payload", func(t *testing.T) {
		maxPayload := make([]byte, MaxPayloadLength)
		if _, err := EncodePacket(maxPayload); err != nil {
			t.Fatalf("EncodePacket at max: %v", err)
		}
		tooLarge := make([]byte, MaxPayloadLength+1)
		if _, err := EncodePacket(tooLarge); !errors.Is(err, ErrPayloadTooLarge) {
			t.Fatalf("got %v, want ErrPayloadTooLarge", err)
		}
	})
}

func TestDecodePacketValidation(t *testing.T) {
	validPayload := []byte{0x01, 0x02, 0x03}
	validPkt, err := EncodePacket(validPayload)
	if err != nil {
		t.Fatalf("EncodePacket: %v", err)
	}

	tests := []struct {
		name    string
		input   []byte
		wantErr error
	}{
		{
			name:    "too short buffer",
			input:   []byte{0x03, 0x00, 0x00},
			wantErr: ErrTooShort,
		},
		{
			name: "invalid version",
			input: func() []byte {
				cp := append([]byte(nil), validPkt...)
				cp[0] = 0x02
				return cp
			}(),
			wantErr: ErrInvalidVersion,
		},
		{
			name: "declared length below minimum",
			input: func() []byte {
				cp := append([]byte(nil), validPkt...)
				cp[2], cp[3] = 0x00, 0x06
				return cp
			}(),
			wantErr: ErrInvalidPacketLength,
		},
		{
			name: "length mismatch larger declaration",
			input: func() []byte {
				cp := append([]byte(nil), validPkt...)
				cp[3]++
				return cp
			}(),
			wantErr: ErrLengthMismatch,
		},
		{
			name: "trailing bytes rejected",
			input: func() []byte {
				return append(append([]byte(nil), validPkt...), 0x00)
			}(),
			wantErr: ErrLengthMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePacket(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DecodePacket() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecodePacketIgnoresReserved(t *testing.T) {
	for _, reserved := range []byte{0x00, 0x01, 0x80, 0xff} {
		pkt := []byte{Version, reserved, 0x00, 0x07, 'a', 'b', 'c'}
		got, err := DecodePacket(pkt)
		if err != nil {
			t.Fatalf("reserved=%#x: unexpected error %v", reserved, err)
		}
		if !bytes.Equal(got, []byte{'a', 'b', 'c'}) {
			t.Fatalf("reserved=%#x: payload %v", reserved, got)
		}
	}
}

func TestEncodePacketAlwaysZeroReserved(t *testing.T) {
	pkt, err := EncodePacket([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatal(err)
	}
	if pkt[1] != 0 {
		t.Fatalf("reserved = %d, want 0", pkt[1])
	}
}

func TestDecodePacketAliasing(t *testing.T) {
	pkt, err := EncodePacket([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodePacket(pkt)
	if err != nil {
		t.Fatal(err)
	}
	decoded[0] ^= 0xff
	if pkt[HeaderLength] != decoded[0] {
		t.Fatal("expected DecodePacket payload to alias packet buffer")
	}
}

func TestDecodeHeaderLengthEncoding(t *testing.T) {
	payload := make([]byte, 100)
	for i := range payload {
		payload[i] = byte(i)
	}
	pkt, err := EncodePacket(payload)
	if err != nil {
		t.Fatal(err)
	}
	declared := int(binary.BigEndian.Uint16(pkt[2:4]))
	if declared != len(pkt) {
		t.Fatalf("declared length %d != len(pkt) %d", declared, len(pkt))
	}
}

func TestDecodeHeaderTooShort(t *testing.T) {
	// DecodePacket rejects short buffers before decodeHeader; exercise the
	// defensive guard directly.
	_, err := decodeHeader([]byte{Version, 0x00, 0x00})
	if !errors.Is(err, ErrTooShort) {
		t.Fatalf("got %v, want ErrTooShort", err)
	}
}
