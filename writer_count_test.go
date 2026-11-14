// SPDX-License-Identifier: MIT

package tpkt

import (
	"errors"
	"testing"
)

type countWriter struct {
	n     int
	calls int
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.calls++
	return c.n, nil
}

func TestWriterInvalidCount(t *testing.T) {
	for _, n := range []int{-1, 100} {
		cw := &countWriter{n: n}
		w, err := NewWriter(cw)
		if err != nil {
			t.Fatal(err)
		}
		err = w.WritePacket([]byte{1, 2, 3})
		if !errors.Is(err, errInvalidWriteCount) {
			t.Fatalf("n=%d: got %v, want errInvalidWriteCount", n, err)
		}
		if cw.calls != 1 {
			t.Fatalf("n=%d: %d writes, want 1", n, cw.calls)
		}
		err = w.WritePacket([]byte{1, 2, 3})
		if !errors.Is(err, ErrStreamMisaligned) || !errors.Is(err, errInvalidWriteCount) {
			t.Fatalf("n=%d: second write got %v, want misaligned", n, err)
		}
		if cw.calls != 1 {
			t.Fatalf("n=%d: stream written after failure", n)
		}
	}
}
