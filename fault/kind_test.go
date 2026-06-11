package fault

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestKindOfBuiltinFaults(t *testing.T) {
	tests := []struct {
		name string
		f    Fault
		want Kind
	}{
		{"latency", Latency(time.Millisecond), KindLatency},
		{"jittered", Jittered(time.Millisecond, 2*time.Millisecond), KindJittered},
		{"jittered_seed", JitteredSeed(time.Millisecond, 2*time.Millisecond, 1), KindJittered},
		{"error", Error(errors.New("x")), KindError},
		{"panic", Panic("boom"), KindPanic},
		{"conn_drop", ConnDrop(), KindConnDrop},
		{"clock", Clock(time.Hour), KindClock},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KindOf(tt.f); got != tt.want {
				t.Fatalf("KindOf(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// customFault is a user-defined fault that does not implement Kinded.
type customFault struct{}

func (customFault) Apply(context.Context) error { return nil }

func TestKindOfCustomFaultIsUnknown(t *testing.T) {
	if got := KindOf(customFault{}); got != KindUnknown {
		t.Fatalf("KindOf(custom) = %v, want KindUnknown", got)
	}
}
