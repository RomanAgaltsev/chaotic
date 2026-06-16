package fault

import (
	"context"
	"strings"
	"testing"
	"time"
)

// durationer mirrors the interface engine.Eval uses to read a fault's nominal
// duration for observer latency metrics without running it.
type durationer interface {
	Duration() time.Duration
}

func TestFaultDurationIntrospection(t *testing.T) {
	cases := []struct {
		name string
		f    Fault
		want time.Duration
	}{
		{"latency", Latency(2 * time.Second), 2 * time.Second},
		{"jittered reports ceiling", Jittered(1*time.Second, 5*time.Second), 5 * time.Second},
		{"seeded jitter reports ceiling", JitteredSeed(1*time.Second, 5*time.Second, 7), 5 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := tc.f.(durationer)
			if !ok {
				t.Fatalf("%T does not expose Duration()", tc.f)
			}
			if got := d.Duration(); got != tc.want {
				t.Fatalf("Duration() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSeededJitterApply(t *testing.T) {
	// A non-empty window exercises the random draw; an equal window exercises
	// the degenerate branch that returns min without locking the RNG. Both use
	// nanosecond durations so the sleep is effectively instant.
	for _, f := range []Fault{
		JitteredSeed(1, 100, 42),
		JitteredSeed(5, 5, 42),
	} {
		if err := f.Apply(context.Background()); err != nil {
			t.Fatalf("Apply() = %v, want nil", err)
		}
	}
}

func TestSentinelErrorStrings(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"http status", &HTTPStatusError{Code: 503}, "chaotic: http status 503"},
		{"header mutation", &HeaderFaultError{Key: "X", Value: "y"}, "chaotic: header mutation"},
		{"stream truncate", &StreamFaultError{Mode: StreamTruncate, Limit: 4}, "truncate 4 B"},
		{"stream slow write", &StreamFaultError{Mode: StreamSlowWrite, Rate: 512}, "slow write 512 B/s"},
		{"stream slow read", &StreamFaultError{Mode: StreamSlowRead, Rate: 1024}, "slow read 1024 B/s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); !strings.Contains(got, tc.want) {
				t.Fatalf("Error() = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}
