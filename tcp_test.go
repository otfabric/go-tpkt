// SPDX-License-Identifier: MIT

package tpkt

import (
	"bytes"
	"net"
	"sync"
	"testing"
	"time"
)

func TestTCPMultiPacketSmoke(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	payloads := [][]byte{
		{0x01, 0x02, 0x03},
		{0xaa, 0xbb, 0xcc, 0xdd},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			t.Errorf("Accept: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		r, err := NewReader(conn, ReaderConfig{})
		if err != nil {
			t.Errorf("NewReader: %v", err)
			return
		}
		for i, want := range payloads {
			got, err := r.ReadPacket()
			if err != nil {
				t.Errorf("ReadPacket %d: %v", i, err)
				return
			}
			if !bytes.Equal(got, want) {
				t.Errorf("packet %d: got %v want %v", i, got, want)
			}
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	w, err := NewWriter(conn)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range payloads {
		if err := w.WritePacket(p); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
}

func TestTCPFullDuplexSmoke(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	req := []byte{0x10, 0x20, 0x30}
	resp := []byte{0x40, 0x50, 0x60}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			t.Errorf("Accept: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		r, err := NewReader(conn, ReaderConfig{})
		if err != nil {
			t.Errorf("NewReader: %v", err)
			return
		}
		w, err := NewWriter(conn)
		if err != nil {
			t.Errorf("NewWriter: %v", err)
			return
		}
		got, err := r.ReadPacket()
		if err != nil {
			t.Errorf("ReadPacket: %v", err)
			return
		}
		if !bytes.Equal(got, req) {
			t.Errorf("got %v want %v", got, req)
			return
		}
		if err := w.WritePacket(resp); err != nil {
			t.Errorf("WritePacket: %v", err)
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	w, err := NewWriter(conn)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(conn, ReaderConfig{})
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
		t.Fatalf("got %v want %v", got, resp)
	}
	wg.Wait()
}
