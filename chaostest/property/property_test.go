package property

import (
	"errors"
	"math/rand/v2"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// genNamed returns a RuleGen that always produces a rule named name (ignoring
// randomness), so tests can reason about which generators are present.
func genNamed(name string) RuleGen {
	return func(_ *rand.Rand) engine.Rule {
		return engine.NewRule(engine.MatchKind(engine.OpHTTPClient)).Named(name)
	}
}

func TestRunPassesWhenInvariantHolds(t *testing.T) {
	gens := []RuleGen{genNamed("a"), genNamed("b")}
	res := run(gens, func(*engine.Engine) error { return nil }, config{runs: 50, seed: 1})
	if res.failed {
		t.Fatalf("run reported a failure for an always-passing body: %+v", res)
	}
}

func TestRunMinimizesToTheCulpritGenerator(t *testing.T) {
	gens := []RuleGen{genNamed("a"), genNamed("culprit"), genNamed("c")}
	// The invariant fails iff the "culprit" rule is present in the engine.
	body := func(eng *engine.Engine) error {
		if eng.AllHits() == nil { // unreachable; keeps eng used
			return nil
		}
		for _, r := range engine.NewRuleSet(snapshotRules(eng)).Snapshot() {
			if r.Name() == "culprit" {
				return errors.New("invariant broken by culprit")
			}
		}
		return nil
	}
	res := run(gens, body, config{runs: 10, seed: 1})
	if !res.failed {
		t.Fatal("run should have found the failing configuration")
	}
	if len(res.minIndices) != 1 || res.minIndices[0] != 1 {
		t.Fatalf("minIndices = %v, want [1] (the culprit generator)", res.minIndices)
	}
}
