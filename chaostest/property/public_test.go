package property

import (
	"math/rand/v2"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// TestPublicTestHoldingInvariant drives the exported Test entry point (and the
// WithRuns/WithSeed options + apply) through its passing path: an invariant that
// always holds must not fail t.
func TestPublicTestHoldingInvariant(t *testing.T) {
	gens := []RuleGen{
		func(*rand.Rand) engine.Rule { return engine.NewRule().Named("a") },
		func(*rand.Rand) engine.Rule { return engine.NewRule().Named("b") },
	}
	holds := func(*engine.Engine) error { return nil }

	Test(t, gens, holds, WithRuns(5), WithSeed(42))
}

// TestApplyClampsRuns confirms the option plumbing: a non-positive run count is
// clamped up to 1 rather than skipping exploration entirely.
func TestApplyClampsRuns(t *testing.T) {
	c := apply([]Option{WithRuns(0), WithSeed(9)})
	if c.runs != 1 {
		t.Errorf("apply clamped runs = %d, want 1", c.runs)
	}
	if c.seed != 9 {
		t.Errorf("apply seed = %d, want 9", c.seed)
	}
}
