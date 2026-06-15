package engine

import "github.com/RomanAgaltsev/chaotic/fault"

// CounterKind classifies a rule's counter for introspection.
type CounterKind int

// Counter kinds, one per rule counter strategy.
const (
	CounterAlways CounterKind = iota
	CounterTimes
	CounterRange
	CounterProbability
	CounterSequence
	CounterStaged
)

// RuleInfo is a read-only view of a Rule for linting and tooling. It exposes
// only what closures permit: whether the rule is unconstrained (no matchers, so
// it matches every Op), its counter kind, and the kinds of its faults.
type RuleInfo struct {
	Name          string
	Unconstrained bool
	Counter       CounterKind
	Faults        []fault.Kind
}

// Info returns a RuleInfo describing r.
func (r Rule) Info() RuleInfo {
	if r.staged != nil {
		var n int
		for _, st := range r.staged.stages {
			n += len(st.Faults)
		}
		fi := make([]fault.Kind, 0, n)
		for _, st := range r.staged.stages {
			for _, f := range st.Faults {
				fi = append(fi, fault.KindOf(f))
			}
		}
		return RuleInfo{
			Name:          r.name,
			Unconstrained: len(r.matchers) == 0,
			Counter:       CounterStaged,
			Faults:        fi,
		}
	}
	fi := make([]fault.Kind, 0, len(r.faults))
	for _, f := range r.faults {
		fi = append(fi, fault.KindOf(f))
	}
	return RuleInfo{
		Name:          r.name,
		Unconstrained: len(r.matchers) == 0,
		Counter:       counterKindOf(r.counter),
		Faults:        fi,
	}
}

func counterKindOf(c counter) CounterKind {
	switch c.(type) {
	case *alwaysCounter:
		return CounterAlways
	case *timesCounter:
		return CounterTimes
	case *rangeCounter:
		return CounterRange
	case *probCounter:
		return CounterProbability
	case *sequenceCounter:
		return CounterSequence
	}
	return CounterAlways
}
