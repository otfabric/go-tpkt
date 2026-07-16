// SPDX-License-Identifier: MIT

package tpkt_test

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/otfabric/go-tpkt"
)

func ExampleEncodePacket() {
	payload := []byte{0x02, 0xf0, 0x80}

	pkt, err := tpkt.EncodePacket(payload)
	if err != nil {
		panic(err)
	}

	version := pkt[0]
	reserved := pkt[1]
	length := int(binary.BigEndian.Uint16(pkt[2:4]))

	fmt.Println(version, reserved, length == len(pkt))
	// Output:
	// 3 0 true
}

func ExampleDecodePacket() {
	pkt := []byte{0x03, 0x00, 0x00, 0x07, 'a', 'b', 'c'}

	decoded, err := tpkt.DecodePacket(pkt)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(decoded))
	// Output:
	// abc
}

func ExampleReader_ReadPacket() {
	payload := []byte{0x01, 0x02, 0x03}
	pkt, _ := tpkt.EncodePacket(payload)

	r, err := tpkt.NewReader(bytes.NewReader(pkt), tpkt.ReaderConfig{})
	if err != nil {
		panic(err)
	}

	got, err := r.ReadPacket()
	if err != nil {
		panic(err)
	}

	fmt.Println(bytes.Equal(got, payload))
	// Output:
	// true
}

func ExampleWriter_WritePacket() {
	var buf bytes.Buffer
	w, err := tpkt.NewWriter(&buf)
	if err != nil {
		panic(err)
	}

	payload := []byte{0x01, 0x02, 0x03}
	if err := w.WritePacket(payload); err != nil {
		panic(err)
	}

	expected, _ := tpkt.EncodePacket(payload)
	fmt.Println(bytes.Equal(buf.Bytes(), expected))
	// Output:
	// true
}
