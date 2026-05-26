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
