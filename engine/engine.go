package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RomanAgaltsev/chaotic/fault"
)

// Option configures an Engine at construction time.
type Option func(*Engine)

// Engine holds the rules and decision logic. Engines are not safe to copy -
// always pass as *Engine. AddRule is safe for concurrent use, Eval is too.
type Engine struct {
	rules        atomic.Pointer[ruleSetHolder]
	observer     Observer
	richObserver RichObserver
	killswitch   KillSwitch
	budget       *failureBudget
	rateLimiter  *tokenBucket
	maxConc      *semaphore
	guard        func() bool

	mu       sync.Mutex               // guards AddRule and hits map mutations
	hits     map[string]*atomic.Int64 // per-named-rule counters
	disabled atomic.Bool
}

// New constructs an engine with no rules.
// Options may inject an Observer or a KillSwitch.
func New(opts ...Option) *Engine {
	e := &Engine{
		hits: make(map[string]*atomic.Int64),
	}
	for _, o := range opts {
		o(e)
	}
	if e.guard != nil && e.guard() {
		panic("chaotic: production guard tripped")
	}
	return e
}

// Disable flips an atomic flag so Enabled reports false and adapters take the
// passthrough path. Faster than Reset for "kill the chaos now". Reversible.
func (e *Engine) Disable() { e.disabled.Store(true) }

// Enable clears the disable flag.
func (e *Engine) Enable() { e.disabled.Store(false) }

// Enabled reports whether the engine has any rules. Adapters call this
// before constructing an Op so the no-op path stays alloc-free.
// Nil-safe: a nil engine reports false.
func (e *Engine) Enabled() bool {
	if e == nil {
		return false
	}
	if e.disabled.Load() {
		return false
	}
	h := e.rules.Load()
	return h != nil && h.rs != nil && h.rs.Len() > 0
}

// AddRule appends a rule. Returns the engine for chaining. Append is
// implemented as a copy-on-write swap of the rule slice so concurrent
// Evals never see a torn slice.
func (e *Engine) AddRule(r Rule) *Engine {
	e.mu.Lock()
	defer e.mu.Unlock()

	var oldRules []Rule
	if h := e.rules.Load(); h != nil {
		oldRules = rulesFor(h)
	}
	newRules := make([]Rule, len(oldRules), len(oldRules)+1)
	copy(newRules, oldRules)
	newRules = append(newRules, r)
	e.rules.Store(&ruleSetHolder{rs: newSliceRuleSet(newRules)})

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
	e.rules.Store(&ruleSetHolder{rs: newSliceRuleSet(nil)})
	e.hits = make(map[string]*atomic.Int64)
}

// ReplaceRules atomically swaps the active rule set. Used by rule sources on
// reload. Concurrent Evals see either the old or the new set, never a torn one.
// Hit counters are rebuilt for the new set's named rules.
func (e *Engine) ReplaceRules(rs RuleSet) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules.Store(&ruleSetHolder{rs: rs})
	e.hits = make(map[string]*atomic.Int64)
	for _, r := range rs.Snapshot() {
		if r.name != "" {
			if _, ok := e.hits[r.name]; !ok {
				e.hits[r.name] = new(atomic.Int64)
			}
		}
	}
}

// WithRuleSource backs the engine with rs at construction (instead of AddRule).
func WithRuleSource(rs RuleSet) Option {
	return func(e *Engine) {
		e.rules.Store(&ruleSetHolder{rs: rs})
		for _, r := range rs.Snapshot() {
			if r.name != "" {
				if _, ok := e.hits[r.name]; !ok {
					e.hits[r.name] = new(atomic.Int64)
				}
			}
		}
	}
}

// Eval evaluates the op against all configured rules and returns the matching
// Action, or Pass if no rule matches or the engine is disabled.
func (e *Engine) Eval(ctx context.Context, op Op) Action {
	if !e.Enabled() {
		return Pass
	}
	if e.killswitch != nil && e.killswitch(ctx, op) {
		return e.passOrTrack()
	}
	h := e.rules.Load()
	for _, r := range rulesFor(h) {
		if !r.matches(ctx, op) {
			continue
		}
		var fire bool
		var faults []fault.Fault
		switch {
		case r.sticky != nil && r.sticky.sticky(op):
			fire, faults = true, r.faults
		case r.staged != nil:
			fire, faults = r.staged.fire()
		case r.counter.shouldFire():
			fire, faults = true, r.faults
			if r.sticky != nil {
				r.sticky.mark(op)
			}
		}
		if !fire {
			e.notifySkip(r.name, op, ReasonCounter)
			continue
		}
		if !fire {
			e.notifySkip(r.name, op, ReasonCounter)
			continue
		}
		if r.rateLimiter != nil && !r.rateLimiter.allow() {
			e.notifySkip(r.name, op, ReasonRateLimit)
			continue
		}
		if e.budget != nil && e.budget.overBudget() {
			e.notifySkip(r.name, op, ReasonFailureBudget)
			return e.passOrTrack()
		}
		if e.rateLimiter != nil && !e.rateLimiter.allow() {
			e.notifySkip(r.name, op, ReasonRateLimit)
			return e.passOrTrack()
		}
		var release func()
		if e.maxConc != nil {
			rel, ok := e.maxConc.tryAcquire()
			if !ok {
				e.notifySkip(r.name, op, ReasonMaxConcurrent)
				return e.passOrTrack()
			}
			release = rel
		}
		e.recordHit(r.name)
		act := &ruleAction{faults: faults, eng: e, op: op, release: release, name: r.name}
		if e.observer != nil && r.name != "" {
			e.observer.RuleFired(r.name, op, act)
		}
		return act
	}
	return e.passOrTrack()
}

// passOrTrack returns the bare Pass singleton unless a failure budget is
// configured, in which case it returns an outcome-tracking action so every
// call's outcome is recorded.
func (e *Engine) passOrTrack() Action {
	if e.budget == nil {
		return Pass
	}
	return &trackingAction{
		eng: e,
	}
}

func (e *Engine) notifySkip(name string, op Op, reason string) {
	if e.observer != nil && name != "" {
		e.observer.RuleSkipped(name, op, reason)
	}
}

func (e *Engine) recordHit(name string) {
	if name == "" {
		return
	}
	e.mu.Lock()
	c := e.hits[name]
	e.mu.Unlock()
	if c != nil {
		c.Add(1)
	}
}

// recordOutcome feeds the wrapped call's result to the failure budget, if any.
func (e *Engine) recordOutcome(callErr error) {
	if e.budget != nil {
		e.budget.record(callErr)
	}
}

// ruleAction runs a rule's faults in order during Before and reports the
// wrapped call's outcome to the engine.
type ruleAction struct {
	faults      []fault.Fault
	eng         *Engine
	op          Op
	name        string
	release     func()
	releaseOnce sync.Once
}

func (a *ruleAction) Before(ctx context.Context) (err error) {
	// Release the concurrency slot if a fault returns an error or panics; on the
	// success path the slot is held until After runs. A Panic fault unwinds
	// straight through here, so recover-release-repanic keeps WithMaxConcurrent
	// honest without altering the panic's propagation.
	defer func() {
		if r := recover(); r != nil {
			a.doRelease()
			panic(r)
		}
		if err != nil {
			a.doRelease()
		}
	}()
	for _, f := range a.faults {
		if err = f.Apply(ctx); err != nil {
			return err
		}
		a.notifyInjected(ctx, f)
	}
	return nil
}

// notifyInjected emits a FaultEvent to the engine's RichObserver, if any, after
// a fault's Apply has run without error. Latency is read from the fault when it
// exposes a Duration method (latency/jittered faults); other faults report 0.
func (a *ruleAction) notifyInjected(ctx context.Context, f fault.Fault) {
	if a.eng == nil || a.eng.richObserver == nil {
		return
	}
	ev := FaultEvent{
		Rule:      a.name,
		Op:        a.op,
		FaultKind: fault.KindOf(f),
	}
	if d, ok := f.(interface{ Duration() time.Duration }); ok {
		ev.Latency = d.Duration()
	}
	a.eng.richObserver.FaultInjected(ctx, ev)
}

func (a *ruleAction) After(_ context.Context) error {
	a.doRelease()
	return nil
}

func (a *ruleAction) Outcome(_ context.Context, callErr error) {
	if a.eng != nil {
		a.eng.recordOutcome(callErr)
	}
}

func (a *ruleAction) doRelease() {
	if a.release != nil {
		a.releaseOnce.Do(a.release)
	}
}

// trackingAction injects nothing. It exists only to record the wrapped
// call's outcome into the failure budget on non-firing paths.
type trackingAction struct {
	eng *Engine
}

func (a *trackingAction) Before(_ context.Context) error {
	return nil
}

func (a *trackingAction) After(_ context.Context) error {
	return nil
}

func (a *trackingAction) Outcome(_ context.Context, callErr error) {
	if a.eng != nil {
		a.eng.recordOutcome(callErr)
	}
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

// WithFailureBudget stops injecting faults once the observed error rate over a
// sliding window of the last window calls reaches maxErrorRate. Requires the
// adapters to report outcomes (they do, via OutcomeReporter). Panics if
// maxErrorRate is outside [0, 1] or window < 1.
func WithFailureBudget(maxErrorRate float64, window int) Option {
	return func(e *Engine) {
		e.budget = newFailureBudget(maxErrorRate, window)
	}
}

// WithRateLimit caps the number of faults that actually fire to rps per second
// (global across all rules). Matched calls beyond the limit return Pass.
func WithRateLimit(rps int) Option {
	return func(e *Engine) {
		e.rateLimiter = newTokenBucket(rps)
	}
}

// WithMaxConcurrent caps the number of simultaneously in-flight faulted calls
// to n. Matched calls that would exceed the cap return Pass. The slot is held
// for the duration of the fault (including latency sleeps) and released when
// the adapter calls After (or when Before short-circuits).
func WithMaxConcurrent(n int) Option {
	return func(e *Engine) {
		e.maxConc = newSemaphore(n)
	}
}

// WithProductionGuard makes New panic if check returns true. Supply a check
// that detects an environment chaos must not run in (e.g. reads an env var).
func WithProductionGuard(check func() bool) Option {
	return func(e *Engine) {
		e.guard = check
	}
}
