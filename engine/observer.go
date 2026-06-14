package engine

import (
	"context"
	"time"

	"github.com/RomanAgaltsev/chaotic/fault"
)

// Skip reasons passed to Observer.RuleSkipped. Observers may switch on these
// instead of matching free-form strings.
const (
	ReasonCounter       = "counter"
	ReasonRateLimit     = "rate_limit"
	ReasonMaxConcurrent = "max_concurrent"
	ReasonFailureBudget = "failure_budget"
	// ReasonDisabled and ReasonKillSwitch are reserved for observers that build
	// their own suppression accounting; the engine's disabled and kill-switch
	// paths return Pass without calling RuleSkipped, so they are not emitted.
	ReasonDisabled   = "disabled"
	ReasonKillSwitch = "killswitch"
)

// Observer receives events from the engine each time it evaluates a named rule.
// v1 ships no concrete implementations. Users supply their own via WithObserver.
// The always-on per-named-rule hit counter (Engine.Hits) does not require
// an observer.
//
// Observer methods are called synchronously on the request path.
// Keep them cheap, do not block.
type Observer interface {
	RuleFired(ruleName string, op Op, action Action)
	RuleSkipped(ruleName string, op Op, reason string)
}

// RichObserver is an optional richer sink. An Observer may also implement it to
// receive per-fault detail the base Observer cannot carry. The engine checks for
// it once, when WithObserver runs, not per call. FaultInjected fires from the
// adapter's request path, synchronously, after a fault's sleep completes - keep
// it cheap and non-blocking, like the base Observer methods.
type RichObserver interface {
	Observer
	FaultInjected(ctx context.Context, ev FaultEvent)
}

// FaultEvent describes a single injected fault delivered to RichObserver. It is
// emitted for faults whose Apply returns without error (latency, jittered, any
// custom no-op fault) and for result mutators (fault.ResponseMutate) after they
// run post-call; faults that short-circuit the call (error, panic, connection
// drop) never produce a FaultEvent.
type FaultEvent struct {
	Rule      string
	Op        Op
	FaultKind fault.Kind
	// Latency is the fault's configured sleep: the exact duration for a Latency
	// fault, or the max bound for a Jittered fault (whose per-call draw is not
	// observable). Zero for faults that inject no sleep.
	Latency time.Duration
}

// KillSwitch lets a caller short-circuit chaos. If it returns true for the
// current Op, Eval returns Pass without consulting any rule. The default
// engine has no kill switch (every call is evaluated).
type KillSwitch func(ctx context.Context, op Op) bool

// WithObserver attaches an Observer to the engine. Pass nil to clear. If obs
// also implements RichObserver, the engine additionally delivers per-fault
// FaultEvents to it; the assertion happens once here, not per call.
func WithObserver(obs Observer) Option {
	return func(e *Engine) {
		e.observer = obs
		if ro, ok := obs.(RichObserver); ok {
			e.richObserver = ro
		} else {
			e.richObserver = nil
		}
	}
}

// WithKillSwitch attaches a kill switch. Pass nil to clear.
func WithKillSwitch(ks KillSwitch) Option {
	return func(e *Engine) {
		e.killswitch = ks
	}
}
