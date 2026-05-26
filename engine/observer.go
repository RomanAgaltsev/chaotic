package engine

import "context"

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

// KillSwitch lets a caller short-circuit chaos. If it returns true for the
// current Op, Eval returns Pass without consulting any rule. The default
// engine has no kill switch (every call is evaluated).
type KillSwitch func(ctx context.Context, op Op) bool

// WithObserver attaches an Observer to the engine. Pass nil to clear.
func WithObserver(obs Observer) Option {
	return func(e *Engine) {
		e.observer = obs
	}
}

// WithKillSwitch attaches a kill switch. Pass nil to clear.
func WithKillSwitch(ks KillSwitch) Option {
	return func(e *Engine) {
		e.killswitch = ks
	}
}
