// SPDX-License-Identifier: MIT

package tpkt_test

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/otfabric/go-tpkt"
)

func ExampleEncode() {
	payload := []byte{0x02, 0xf0, 0x80}

	pkt, err := tpkt.Encode(payload)
	if err != nil {
		panic(err)
	}

	// Inspect header fields of the encoded packet.
	version := pkt[0]
	reserved := pkt[1]
	length := int(binary.BigEndian.Uint16(pkt[2:4]))

	fmt.Println(version, reserved, length == len(pkt))
	// Output:
	// 3 0 true
}

func ExampleDecode() {
	// A minimal valid TPKT packet: version=3, reserved=0, length=7, 3-byte payload.
	pkt := []byte{0x03, 0x00, 0x00, 0x07, 'a', 'b', 'c'}

	decoded, err := tpkt.Decode(pkt)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(decoded))
	// Output:
	// abc
}

func ExampleReader_ReadFrame() {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, _ := tpkt.Encode(payload)

	r := tpkt.NewReader(bytes.NewReader(pkt))

	got, err := r.ReadFrame()
	if err != nil {
		panic(err)
	}

	fmt.Println(bytes.Equal(got, payload))
	// Output:
	// true
}

func ExampleWriter_WriteFrame() {
	var buf bytes.Buffer
	w := tpkt.NewWriter(&buf)

	payload := []byte{0x01, 0x02, 0x03}
	if _, err := w.WriteFrame(payload); err != nil {
		panic(err)
	}

	expected, _ := tpkt.Encode(payload)
	ok := bytes.Equal(buf.Bytes(), expected)
	fmt.Println(ok)
	// Output:
	// true
}
