package fault

import (
	"context"
	"errors"
	"testing"
)

func TestStreamFaultsCarryParams(t *testing.T) {
	tests := []struct {
		name string
		f    Fault
		mode StreamMode
		rate int
		lim  int
		kind Kind
	}{
		{"slow_reader", SlowReader(1024), StreamSlowRead, 1024, 0, KindSlowReader},
		{"slow_writer", SlowWriter(512), StreamSlowWrite, 512, 0, KindSlowWriter},
		{"truncate", Truncate(8), StreamTruncate, 0, 8, KindTruncate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.f.Apply(context.Background())
			var sfe *StreamFaultError
			if !errors.As(err, &sfe) {
				t.Fatalf("Apply = %v, want *StreamFaultError", err)
			}
			if sfe.Mode != tt.mode || sfe.Rate != tt.rate || sfe.Limit != tt.lim {
				t.Fatalf("sentinel = %+v, want mode %v rate %d limit %d", sfe, tt.mode, tt.rate, tt.lim)
			}
			if KindOf(tt.f) != tt.kind {
				t.Fatalf("KindOf = %v, want %v", KindOf(tt.f), tt.kind)
			}
		})
	}
}

func TestStreamFaultFreshSentinelPerApply(t *testing.T) {
	f := Truncate(4)
	a := f.Apply(context.Background())
	b := f.Apply(context.Background())
	var sa, sb *StreamFaultError
	_ = errors.As(a, &sa)
	_ = errors.As(b, &sb)
	if sa == sb {
		t.Fatal("each Apply must return a distinct sentinel pointer")
	}
}
