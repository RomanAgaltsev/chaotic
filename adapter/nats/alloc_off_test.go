//go:build chaos_off

package nats

import (
	"testing"
)

func TestZeroAllocUnderChaosOff(t *testing.T) {
	c := &Conn{conn: &fakeConn{}}
	data := []byte("hi")
	avg := testing.AllocsPerRun(100, func() {
		_ = c.Publish("events", data)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
