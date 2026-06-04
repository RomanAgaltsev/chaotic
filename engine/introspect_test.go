package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/ag4r/chaotic/fault"
)

func TestRuleInfoUnconstrainedForNoMatcherRule(t *testing.T) {
	r := NewRule(WithFault(fault.Panic("boom"))).Named("danger")
	info := r.Info()
	if !info.Unconstrained {
		t.Fatal("rule with no matchers should report Unconstrained")
	}
	if info.Name != "danger" {
		t.Fatalf("Name = %q, want %q", info.Name, "danger")
	}
}

func TestRuleInfoConstrainedWhenMatcherPresent(t *testing.T) {
	r := NewRule(MatchKind(OpHTTPClient), WithFault(fault.Latency(0)))
	if r.Info().Unconstrained {
		t.Fatal("rule with a matcher should not report Unconstrained")
	}
}

func TestRuleInfoListsFaultKindsInOrder(t *testing.T) {
	r := NewRule(WithFaults(
		fault.Latency(time.Millisecond),
		fault.Error(errors.New("x")),
	))
	got := r.Info().Faults
	want := []fault.Kind{fault.KindLatency, fault.KindError}
	if len(got) != len(want) {
		t.Fatalf("Faults = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Faults[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRuleInfoCounterKind(t *testing.T) {
	tests := []struct {
		name string
		opt  RuleOption
		want CounterKind
	}{
		{"always_default", nil, CounterAlways},
		{"times", Times(3), CounterTimes},
		{"range", Range(1, 2), CounterRange},
		{"probability", Probability(0.5, 1), CounterProbability},
		{"sequence", Sequence([]bool{true}), CounterSequence},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []RuleOption{WithFault(fault.Latency(0))}
			if tt.opt != nil {
				opts = append(opts, tt.opt)
			}
			if got := NewRule(opts...).Info().Counter; got != tt.want {
				t.Fatalf("Counter = %v, want %v", got, tt.want)
			}
		})
	}
}
