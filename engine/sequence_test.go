package engine

import (
	"testing"

	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestSequenceFiresOnMaskedPositions(t *testing.T) {
	r := NewRule(MatchKind(OpExplicit), Sequence([]bool{true, false, true}), WithFault(fault.Latency(0)))
	got := make([]bool, 0, 5)
	for range 5 {
		got = append(got, r.counter.shouldFire())
	}
	want := []bool{true, false, true, false, false} // exhausted => false
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("fire[%d] = %v, want %v (seq %v)", i, got[i], want[i], got)
		}
	}
}

func TestSequenceEmptyNeverFires(t *testing.T) {
	r := NewRule(Sequence(nil), WithFault(fault.Latency(0)))
	if r.counter.shouldFire() {
		t.Fatal("empty sequence should never fire")
	}
}
