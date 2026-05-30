package engine

// RuleSet is the engine's view of its current rules.
// The engine never holds a snapshot across Eval calls - it loads
// a fresh one each time.
type RuleSet interface {
	Len() int
	Snapshot() []Rule
}

// sliceRuleSet is the in-memory implementation. Stored by Engine behind an
// atomic.Pointer so AddRule can swap it without locking the hot path.
type sliceRuleSet struct {
	rules []Rule
}

func newSliceRuleSet(rules []Rule) *sliceRuleSet {
	return &sliceRuleSet{
		rules: rules,
	}
}

func (rs *sliceRuleSet) Len() int {
	return len(rs.rules)
}

// Snapshot returns a defensive copy so callers may freely mutate the slice.
// The Rule values inside still share their pointer-backed counters.
func (rs *sliceRuleSet) Snapshot() []Rule {
	out := make([]Rule, len(rs.rules))
	copy(out, rs.rules)
	return out
}

// NewRuleSet returns an in-memory RuleSet backed by the given rules.
// Sources (file/http) build their rules then call ReplaceRules(NewRuleSet(rules)).
func NewRuleSet(rules []Rule) RuleSet {
	return newSliceRuleSet(rules)
}

// ruleLister is an unexported fast path: Eval uses it to iterate without the
// defensive copy that Shapshot makes. sliceRuleSet implements it.
type ruleLister interface {
	rulesForEval() []Rule
}

func (s *sliceRuleSet) rulesForEval() []Rule {
	return s.rules
}

// ruleSetHolder lets the engine store a RuleSet interface inside
// an atomic.Pointer (which requires a concrete element type).
type ruleSetHolder struct {
	rs RuleSet
}

// rulesFor extracts the rules for iteration: the no-copy fast path when the
// set implements ruleLister, else a Snapshot.
func rulesFor(h *ruleSetHolder) []Rule {
	if h == nil || h.rs == nil {
		return nil
	}
	if rl, ok := h.rs.(ruleLister); ok {
		return rl.rulesForEval()
	}
	return h.rs.Snapshot()
}
