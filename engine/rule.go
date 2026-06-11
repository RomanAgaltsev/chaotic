package engine

import (
	"context"
	"math/rand/v2"
	"path"
	"sync"
	"sync/atomic"

	"github.com/ag4r/chaotic/fault"
)

// RuleOption configures a Rule during construction.
type RuleOption func(*Rule)

// Rule is a single match-counter-faults triple. Construct with NewRule and
// pass to Engine.AddRule. Rule values may be copied (e.g., by Named), but
// they share state via internal pointers - never modify a Rule's selectors or
// faults after passing it to AddRule.
type Rule struct {
	name        string
	matchers    []matcher
	counter     counter
	faults      []fault.Fault
	sticky      *stickyTracker
	rateLimiter *tokenBucket
}

type matcher func(ctx context.Context, op Op) bool

// NewRule constructs a Rule. The default counter is Always, default action is
// no faults (Pass).
func NewRule(opts ...RuleOption) Rule {
	r := Rule{counter: &alwaysCounter{}}
	for _, o := range opts {
		o(&r)
	}
	return r
}

// matches reports whether all selectors are satisfied. A Rule with no
// selectors matches everything.
func (r *Rule) matches(ctx context.Context, op Op) bool {
	for _, m := range r.matchers {
		if !m(ctx, op) {
			return false
		}
	}
	return true
}

// MatchKind matches Ops whose Kind appears in kinds. With zero arguments,
// matches nothing.
func MatchKind(kinds ...Kind) RuleOption {
	ks := append([]Kind(nil), kinds...)
	return func(r *Rule) {
		r.matchers = append(r.matchers, func(_ context.Context, op Op) bool {
			for _, k := range ks {
				if op.Kind == k {
					return true
				}
			}
			return false
		})
	}
}

// MatchName matches Ops whose Name satisfies path.Match(pattern, Name).
// Patterns support *, ?, and [...]. * does not cross /.
func MatchName(pattern string) RuleOption {
	return func(r *Rule) {
		r.matchers = append(r.matchers, func(_ context.Context, op Op) bool {
			ok, err := path.Match(pattern, op.Name)
			return err == nil && ok
		})
	}
}

// MatchAttr matches Ops whose Attrs[key] equals value.
func MatchAttr(key, value string) RuleOption {
	return func(r *Rule) {
		r.matchers = append(r.matchers, func(_ context.Context, op Op) bool {
			v, ok := op.Attrs[key]
			return ok && v == value
		})
	}
}

// MatchPredicate matches Ops for which fn returns true.
func MatchPredicate(fn func(context.Context, Op) bool) RuleOption {
	return func(r *Rule) {
		r.matchers = append(r.matchers, fn)
	}
}

// counter decides whether a matched Op should actually fire the rule's
// faults. Counters are stateful and shared across copies of a Rule via
// pointers - see timesCounter et al.
type counter interface {
	shouldFire() bool
}

type alwaysCounter struct{}

func (a *alwaysCounter) shouldFire() bool {
	return true
}

// Always is default counter: every match fires the rule.
func Always() RuleOption {
	return func(r *Rule) {
		r.counter = &alwaysCounter{}
	}
}

// Times makes the rule fire on the first n matches. After n, the rule never
// fires again until Engine.Reset clears the counter.
func Times(n int) RuleOption {
	return func(r *Rule) {
		r.counter = &timesCounter{
			n: int64(n),
		}
	}
}

type timesCounter struct {
	n   int64
	cur atomic.Int64
}

func (t *timesCounter) shouldFire() bool {
	return t.cur.Add(1) <= t.n
}

// Range makes the rule fire on matches[from:to] (1-indexed, inclusive).
// If from > to or either is < 1, the rule never fires.
func Range(from, to int) RuleOption {
	return func(r *Rule) {
		r.counter = &rangeCounter{from: int64(from), to: int64(to)}
	}
}

type rangeCounter struct {
	from, to int64
	cur      atomic.Int64
}

func (r *rangeCounter) shouldFire() bool {
	n := r.cur.Add(1)
	return n >= r.from && n <= r.to
}

// Probability makes the rule fire on each match independently with
// probability p. Seed makes the decision deterministic across runs.
// Panics if p is outside [0,1].
func Probability(p float64, seed int64) RuleOption {
	if p < 0 || p > 1 {
		panic("chaotic: Probability p must be in [0,1]")
	}
	return func(r *Rule) {
		r.counter = &probCounter{
			p:   p,
			rng: rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9E3779B97F4A7C15)), //nolint:gosec // non-cryptographic randomness is intentional for chaos probability decisions
		}
	}
}

type probCounter struct {
	p   float64
	mu  sync.Mutex
	rng *rand.Rand
}

func (p *probCounter) shouldFire() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rng.Float64() < p.p
}

// Sequence fires on the evaluations whose index is true in fire, in order, and
// skips the rest. After the slice is exhausted it never fires again. it is the
// deterministic counter chaostest/golden uses to replay a recorded fire
// pattern, and is useful generally for "fire on exactly these matches".
func Sequence(fire []bool) RuleOption {
	mask := append([]bool(nil), fire...)
	return func(r *Rule) {
		r.counter = &sequenceCounter{fire: mask}
	}
}

type sequenceCounter struct {
	fire []bool
	cur  atomic.Int64
}

func (s *sequenceCounter) shouldFire() bool {
	i := s.cur.Add(1) - 1
	if i < 0 || int(i) >= len(s.fire) {
		return false
	}
	return s.fire[i]
}

// WithFault attaches a single fault. Equivalent to WithFaults(f).
func WithFault(f fault.Fault) RuleOption {
	return func(r *Rule) {
		r.faults = append(r.faults, f)
	}
}

// WithFaults attaches faults that execute in order inside Action.Before.
// The first fault returning a non-nil error short-circuits the chain.
func WithFaults(fs ...fault.Fault) RuleOption {
	return func(r *Rule) {
		r.faults = append(r.faults, fs...)
	}
}

// Named tags the rule for assertions and observability. Returns a copy of
// the rule (with shared mutable state - selectors/counter/faults) so the
// pre-named rule remains usable.
func (r Rule) Named(name string) Rule {
	r.name = name
	return r
}

// Name reports the rule's name, or "" if it has none.
func (r Rule) Name() string {
	return r.name
}
