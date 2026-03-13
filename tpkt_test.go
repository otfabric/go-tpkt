package tpkt

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name:    "minimum TPDU length (3 bytes)",
			payload: []byte{0x01, 0x02, 0x03},
		},
		{
			name:    "small payload",
			payload: []byte{0xde, 0xad, 0xbe, 0xef},
		},
		{
			name:    "arbitrary payload",
			payload: []byte("hello, tpkt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt, err := Encode(tt.payload)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			got, err := Decode(pkt)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if !bytes.Equal(got, tt.payload) {
				t.Fatalf("payload mismatch: got %v want %v", got, tt.payload)
			}
		})
	}
}

func TestEncodeLengthValidation(t *testing.T) {
	t.Run("empty payload rejected", func(t *testing.T) {
		// total length would be HeaderLength (4), below RFC 1006 minimum of 7.
		if _, err := Encode(nil); !errors.Is(err, ErrInvalidLength) {
			t.Fatalf("Encode() with empty payload: want ErrInvalidLength, got %v", err)
		}
	})

	t.Run("payload smaller than 3 bytes rejected", func(t *testing.T) {
		tooShort := []byte{0x01, 0x02}
		if _, err := Encode(tooShort); !errors.Is(err, ErrInvalidLength) {
			t.Fatalf("Encode() with too short payload: want ErrInvalidLength, got %v", err)
		}
	})

	t.Run("max size boundary", func(t *testing.T) {
		maxPayload := make([]byte, rfcMaxPacketLength-HeaderLength)
		if _, err := Encode(maxPayload); err != nil {
			t.Fatalf("Encode() at max size error = %v", err)
		}

		tooLarge := make([]byte, rfcMaxPacketLength-HeaderLength+1)
		if _, err := Encode(tooLarge); !errors.Is(err, ErrFrameTooLarge) {
			t.Fatalf("Encode() above max size: want ErrFrameTooLarge, got %v", err)
		}
	})
}

func TestDecodeValidationErrors(t *testing.T) {
	validPayload := []byte{0x01, 0x02, 0x03}
	validPkt, err := Encode(validPayload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	tests := []struct {
		name    string
		mutate  func([]byte) []byte
		wantErr error
	}{
		{
			name: "too short buffer",
			mutate: func(_ []byte) []byte {
				return []byte{0x03, 0x00, 0x00}
			},
			wantErr: ErrTooShort,
		},
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
			name: "declared length smaller than minimum",
			mutate: func(b []byte) []byte {
				cp := append([]byte(nil), b...)
				// Force a value below rfcMinPacketLength.
				cp[2] = 0x00
				cp[3] = 0x06
				return cp
			},
			wantErr: ErrInvalidLength,
		},
		{
			name: "length mismatch larger than buffer",
			mutate: func(b []byte) []byte {
				cp := append([]byte(nil), b...)
				// Increase declared length by 1.
				cp[3]++
				return cp
			},
			wantErr: ErrLengthMismatch,
		},
		{
			name: "length mismatch smaller than buffer",
			mutate: func(b []byte) []byte {
				// Append one extra trailing byte so actual > declared.
				cp := append([]byte(nil), b...)
				cp = append(cp, 0x00)
				return cp
			},
			wantErr: ErrLengthMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.mutate(validPkt)
			_, err := Decode(input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Decode() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseFrame(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	frame, err := Parse(pkt)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if frame.Len() != len(pkt) {
		t.Fatalf("Frame.Len() = %d, want %d", frame.Len(), len(pkt))
	}
	if !bytes.Equal(frame.Payload, payload) {
		t.Fatalf("Frame.Payload = %v, want %v", frame.Payload, payload)
	}

	// MarshalBinary should round-trip.
	gotPkt, err := frame.MarshalBinary()
	if err != nil {
		t.Fatalf("Frame.MarshalBinary() error = %v", err)
	}
	if !bytes.Equal(gotPkt, pkt) {
		t.Fatalf("MarshalBinary() mismatch: got %v want %v", gotPkt, pkt)
	}
}

func TestFrameMarshalBinaryInvalidPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		wantErr error
	}{
		{
			name:    "empty payload",
			payload: nil,
			wantErr: ErrInvalidLength,
		},
		{
			name:    "too short payload",
			payload: []byte{0x01, 0x02},
			wantErr: ErrInvalidLength,
		},
		{
			name:    "too large payload",
			payload: make([]byte, MaxPacketLength-HeaderLength+1),
			wantErr: ErrFrameTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Frame{Payload: tt.payload}
			_, err := f.MarshalBinary()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Frame.MarshalBinary() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeAliasing(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	decoded, err := Decode(pkt)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	decoded[0] ^= 0xff
	if pkt[HeaderLength] != decoded[0] {
		t.Fatalf("expected Decode() payload to alias packet buffer")
	}
}

func TestParseAliasing(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	frame, err := Parse(pkt)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	frame.Payload[1] ^= 0xff
	if pkt[HeaderLength+1] != frame.Payload[1] {
		t.Fatalf("expected Frame.Payload to alias packet buffer")
	}
}
