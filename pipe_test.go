// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestPipeRoundTrip(t *testing.T) {
	c1, c2 := net.Pipe()
	defer func() { _ = c1.Close() }()
	defer func() { _ = c2.Close() }()
	deadline := time.Now().Add(5 * time.Second)
	_ = c1.SetDeadline(deadline)
	_ = c2.SetDeadline(deadline)

	payload := []byte{0x01, 0x02, 0x03}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w, err := NewWriter(c1)
		if err != nil {
			t.Errorf("NewWriter: %v", err)
			return
		}
		if err := w.WritePacket(payload); err != nil {
			t.Errorf("WritePacket: %v", err)
		}
	}()

	r, err := NewReader(c2, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadPacket()
	if err != nil {
		t.Fatalf("ReadPacket: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %v want %v", got, payload)
	}
	wg.Wait()
}

func TestPipeMultiPacket(t *testing.T) {
	c1, c2 := net.Pipe()
	defer func() { _ = c1.Close() }()
	defer func() { _ = c2.Close() }()
	deadline := time.Now().Add(5 * time.Second)
	_ = c1.SetDeadline(deadline)
	_ = c2.SetDeadline(deadline)

	payloads := [][]byte{
		{0x01, 0x02, 0x03},
		{0x0a, 0x0b, 0x0c, 0x0d},
		{0x11, 0x22, 0x33},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() { _ = c1.Close() }()
		w, err := NewWriter(c1)
		if err != nil {
			t.Errorf("NewWriter: %v", err)
			return
		}
		for _, p := range payloads {
			if err := w.WritePacket(p); err != nil {
				t.Errorf("WritePacket: %v", err)
				return
			}
		}
	}()

	r, err := NewReader(c2, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range payloads {
		got, err := r.ReadPacket()
		if err != nil {
			t.Fatalf("packet %d: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("packet %d: got %v want %v", i, got, want)
		}
	}
	if _, err := r.ReadPacket(); !errors.Is(err, io.EOF) {
		t.Fatalf("after close: %v, want EOF", err)
	}
	wg.Wait()
}

func TestPipeFullDuplex(t *testing.T) {
	c1, c2 := net.Pipe()
	defer func() { _ = c1.Close() }()
	defer func() { _ = c2.Close() }()
	deadline := time.Now().Add(5 * time.Second)
	_ = c1.SetDeadline(deadline)
	_ = c2.SetDeadline(deadline)

	req := []byte{0xaa, 0xbb, 0xcc}
	resp := []byte{0x11, 0x22, 0x33}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := NewReader(c2, ReaderConfig{})
		if err != nil {
			t.Errorf("server NewReader: %v", err)
			return
		}
		w, err := NewWriter(c2)
		if err != nil {
			t.Errorf("server NewWriter: %v", err)
			return
		}
		got, err := r.ReadPacket()
		if err != nil {
			t.Errorf("server ReadPacket: %v", err)
			return
		}
		if !bytes.Equal(got, req) {
			t.Errorf("server got %v want %v", got, req)
			return
		}
		if err := w.WritePacket(resp); err != nil {
			t.Errorf("server WritePacket: %v", err)
		}
	}()

	w, err := NewWriter(c1)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(c1, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WritePacket(req); err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, resp) {
		t.Fatalf("client got %v want %v", got, resp)
	}
	wg.Wait()
}

func TestPipeAdversarialBadVersion(t *testing.T) {
	c1, c2 := net.Pipe()
	defer func() { _ = c1.Close() }()
	defer func() { _ = c2.Close() }()
	deadline := time.Now().Add(5 * time.Second)
	_ = c1.SetDeadline(deadline)
	_ = c2.SetDeadline(deadline)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = c1.Write([]byte{0x02, 0x00, 0x00, 0x07, 0x01, 0x02, 0x03})
		_ = c1.Close()
	}()

	r, err := NewReader(c2, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadPacket(); !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("got %v, want ErrInvalidVersion", err)
	}
	wg.Wait()
}

func TestPipeNonZeroReservedAccepted(t *testing.T) {
	c1, c2 := net.Pipe()
	defer func() { _ = c1.Close() }()
	defer func() { _ = c2.Close() }()
	deadline := time.Now().Add(5 * time.Second)
	_ = c1.SetDeadline(deadline)
	_ = c2.SetDeadline(deadline)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = c1.Write([]byte{Version, 0xff, 0x00, 0x07, 0x01, 0x02, 0x03})
		_ = c1.Close()
	}()

	r, err := NewReader(c2, ReaderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadPacket()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte{0x01, 0x02, 0x03}) {
		t.Fatalf("got %v", got)
	}
	wg.Wait()
}
