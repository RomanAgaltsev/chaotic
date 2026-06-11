//go:build chaos_off

package net_test

import (
	"net"
	"testing"

	chaosnet "github.com/ag4r/chaotic/adapter/net"
	"github.com/ag4r/chaotic/engine"
)

func TestZeroAllocUnderChaosOff(t *testing.T) {
	a, b := net.Pipe()
	t.Cleanup(func() { _ = a.Close(); _ = b.Close() })
	eng := engine.New()
	avg := testing.AllocsPerRun(100, func() {
		_ = chaosnet.WrapConn(a, eng)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off (WrapConn returns the conn unchanged)", avg)
	}
}
