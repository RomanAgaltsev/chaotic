package engine

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ag4r/chaotic/fault"
)

// Option configures an Engine at construction time.
type Option func(*Engine)

// Engine holds the rules and decision logic. Engines are not safe to copy -
// always pass as *Engine. AddRule is safe for concurrent use, Eval is too.
type Engine struct {
	rules      atomic.Pointer[sliceRuleSet]
	observer   Observer
	killswitch KillSwitch

	mu   sync.Mutex               // guards AddRule and hits map mutations
	hits map[string]*atomic.Int64 // per-named-rule counters
}

// New constructs an engine with no rules.
// Options may inject an Observer, a KillSwitch, or other.
func New(opts ...Option) *Engine {
	e := &Engine{
		hits: make(map[string]*atomic.Int64),
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// Enabled reports whether the engine has any rules. Adapters call this
// before constructing an Op so the no-op path stays alloc-free.
// Nil-safe: a nil engine reports false.
func (e *Engine) Enabled() bool {
	if e == nil {
		return false
	}
	rs := e.rules.Load()
	return rs != nil && rs.Len() > 0
}

// AddRule appends a rule. Returns the engine for chaining. Append is
// implemented as a copy-on-write swap of the rule slice so concurrent
// Evals never see a torn slice.
func (e *Engine) AddRule(r Rule) *Engine {
	e.mu.Lock()
	defer e.mu.Unlock()

	var oldRules []Rule
	if rs := e.rules.Load(); rs != nil {
		oldRules = rs.rules
	}
	newRules := make([]Rule, len(oldRules), len(oldRules)+1)
	copy(newRules, oldRules)
	newRules = append(newRules, r)
	e.rules.Store(newSliceRuleSet(newRules))

	if r.name != "" {
		if _, ok := e.hits[r.name]; !ok {
			e.hits[r.name] = new(atomic.Int64)
		}
	}

	return e
}

// Reset clears all rules and hit counters. Idempotent and cheap.
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules.Store(newSliceRuleSet(nil))
	e.hits = make(map[string]*atomic.Int64)
}

func (e *Engine) Eval(ctx context.Context, op Op) Action {
	if !e.Enabled() {
		return Pass
	}
	if e.killswitch != nil && e.killswitch(ctx, op) {
		return Pass
	}
	rs := e.rules.Load()
	for _, r := range rs.rules {
		if !r.matches(ctx, op) {
			continue
		}
		if !r.counter.shouldFire() {
			if e.observer != nil && r.name != "" {
				e.observer.RuleSkipped(r.name, op, "counter")
			}
			continue
		}
		// Hit: increment counter, build action, notify observer, return.
		if r.name != "" {
			e.mu.Lock()
			c := e.hits[r.name]
			e.mu.Unlock()
			if c != nil {
				c.Add(1)
			}
		}
		act := ruleAction{
			faults: r.faults,
		}
		if e.observer != nil && r.name != "" {
			e.observer.RuleFired(r.name, op, act)
		}
		return act
	}
	return Pass
}

// ruleAction runs a rule's faults in order during Before.
type ruleAction struct {
	faults []fault.Fault
}

func (a ruleAction) Before(ctx context.Context) error {
	for _, f := range a.faults {
		if err := f.Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a ruleAction) After(ctx context.Context) error {
	return nil
}

// Hits returns the number of times a named rule has fired.
// Unknown names return 0. Safe for concurrent use.
func (e *Engine) Hits(name string) int {
	if e == nil || name == "" {
		return 0
	}
	e.mu.Lock()
	c := e.hits[name]
	e.mu.Unlock()
	if c == nil {
		return 0
	}
	return int(c.Load())
}

// AllHits returns a snapshot of hit counts for every named rule registered
// with the engine, including rules that have not yet fired (value 0).
func (e *Engine) AllHits() map[string]int {
	if e == nil {
		return map[string]int{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make(map[string]int, len(e.hits))
	for name, c := range e.hits {
		out[name] = int(c.Load())
	}
	return out
}
