package property

import (
	"flag"
	"math/rand/v2"
	"testing"

	"github.com/ag4r/chaotic/engine"
)

// RuleGen produces a randomized Rule from r. It is the search space for a
// property test.
type RuleGen func(r *rand.Rand) engine.Rule

// Option configures a property run.
type Option func(*config)

type config struct {
	runs int
	seed int64
}

var runsFlag = flag.Int("chaos-property-runs", 100,
	"number of randomized configurations chaostest/property explores")

func apply(opts []Option) config {
	c := config{runs: *runsFlag, seed: 1}
	for _, o := range opts {
		o(&c)
	}
	if c.runs < 1 {
		c.runs = 1
	}
	return c
}

// WithRuns sets the number of randomized configurations to explore.
func WithRuns(n int) Option { return func(c *config) { c.runs = n } }

// WithSeed sets the base seed, making a run reproducible (e.g. to replay a
// reported counterexample with WithRuns(1)).
func WithSeed(seed int64) Option { return func(c *config) { c.seed = seed } }

// result is run's outcome: whether a failing configuration was found, the seed
// that produced it, and the minimal set of generator indices that still fails.
type result struct {
	failed     bool
	seed       int64
	minIndices []int
}

// rngFor returns the deterministic *rand.Rand for generator index at seed, so
// each generator is independent of the others' draws.
func rngFor(seed int64, index int) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), uint64(index))) //nolint:gosec // -
}

// buildEngine constructs a fresh engine with the rules produced by the given
// generator indices at seed.
func buildEngine(gens []RuleGen, seed int64, indices []int) *engine.Engine {
	eng := engine.New()
	for _, i := range indices {
		eng.AddRule(gens[i](rngFor(seed, i)))
	}
	return eng
}

// snapshotRules returns the rules currently installed on eng (test/inspection
// helper; the engine has no public rule listing).
func snapshotRules(eng *engine.Engine) []engine.Rule {
	// AllHits exposes named rules; for full fidelity the engine would need a
	// public lister. Property only needs names for assertions, which AllHits
	// covers. This helper reconstructs name-only rules for inspection.
	names := eng.AllHits()
	out := make([]engine.Rule, 0, len(names))
	for name := range names {
		out = append(out, engine.NewRule().Named(name))
	}
	return out
}

// run explores up to c.runs configurations. On the first failing one it
// minimizes the generator set and returns the result. A non-failing exploration
// returns result{failed:false}.
func run(gens []RuleGen, body func(*engine.Engine) error, c config) result {
	all := indices(len(gens))
	for i := 0; i < c.runs; i++ {
		seed := c.seed + int64(i)
		if body(buildEngine(gens, seed, all)) != nil {
			return result{failed: true, seed: seed, minIndices: minimize(gens, seed, all, body)}
		}
	}
	return result{}
}

// minimize greedily drops generator indices that are not required for the body
// to keep failing at the given seed (delta debugging).
func minimize(gens []RuleGen, seed int64, start []int, body func(*engine.Engine) error) []int {
	cur := append([]int(nil), start...)
	for changed := true; changed; {
		changed = false
		for _, drop := range cur {
			cand := without(cur, drop)
			if len(cand) == 0 {
				continue
			}
			if body(buildEngine(gens, seed, cand)) != nil {
				cur = cand
				changed = true
				break
			}
		}
	}
	return cur
}

func indices(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func without(s []int, x int) []int {
	out := make([]int, 0, len(s))
	for _, v := range s {
		if v != x {
			out = append(out, v)
		}
	}
	return out
}

// Test explores randomized rule configurations drawn from gens and runs body
// against each. body returns a non-nil error when its invariant is violated for
// that configuration. On the first failure, Test minimizes to the smallest
// failing generator set and fails t with a reproduction recipe.
func Test(t testing.TB, gens []RuleGen, body func(*engine.Engine) error, opts ...Option) {
	t.Helper()
	res := run(gens, body, apply(opts))
	if res.failed {
		t.Errorf("chaostest/property: invariant violated (seed=%d); minimal failing generator indices %v; "+
			"reproduce with property.Test(t, gens, body, property.WithSeed(%d), property.WithRuns(1))",
			res.seed, res.minIndices, res.seed)
	}
}
