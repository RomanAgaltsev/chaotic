//go:build chaos_off

package io_test

import (
	"strings"
	"testing"

	chaosio "github.com/ag4r/chaotic/adapter/io"
	"github.com/ag4r/chaotic/engine"
)

func TestZeroAllocUnderChaosOff(t *testing.T) {
	eng := engine.New()
	r := strings.NewReader("payload")
	avg := testing.AllocsPerRun(100, func() {
		_ = chaosio.WrapReader(r, eng)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off (WrapReader returns the reader unchanged)", avg)
	}
}
