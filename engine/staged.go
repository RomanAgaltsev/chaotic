package engine

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/RomanAgaltsev/chaotic/fault"
)

// Stage is one phase of a staged rule: Times consecutive matches that inject
// Faults in order. Times == 0 means "fire forever" and is legal only as the
// final stage. A stage with empty Faults passes those matches through untouched
// (useful for "let N succeed, then start failing").
type Stage struct {
	Times  int
	Faults []fault.Fault
}

// stagedCounter advances through a rule's stages by cumulative match count.
// One atomic add per match; no locks.
type stagedCounter struct {
	stages    []Stage
	cum       []int64 // cum[i] = cumulative end (1-indexed, inclusive) of finite stage i
	openEnded bool    // final stage has Times == 0
	cur       atomic.Int64
}

// validateStages reports the first structural problem with stages, or nil.
func validateStages(stages []Stage) error {
	if len(stages) == 0 {
		return errors.New("chaotic: WithStages requires at least one stage")
	}
	for i, s := range stages {
		last := i == len(stages)-1
		if !last && s.Times <= 0 {
			return fmt.Errorf("chaotic: stage %d: only the final stage may have Times <= 0 (got %d)", i, s.Times)
		}
		if last && s.Times < 0 {
			return fmt.Errorf("chaotic: final stage Times must be >= 0 (got %d)", s.Times)
		}
	}
	return nil
}

// newStagedCounter assumes stages already passed validateStages.
func newStagedCounter(stages []Stage) *stagedCounter {
	cp := append([]Stage(nil), stages...)
	openEnded := cp[len(cp)-1].Times == 0
	cum := make([]int64, len(cp))
	var total int64
	for i, s := range cp {
		if i == len(cp)-1 && openEnded {
			cum[i] = total // open-ended: no finite end; covered by the openEnded branch in fire
			continue
		}
		total += int64(s.Times)
		cum[i] = total
	}
	return &stagedCounter{stages: cp, cum: cum, openEnded: openEnded}
}

// fire advances by one match and returns whether it fires and the active
// stage's faults.
func (s *stagedCounter) fire() (bool, []fault.Fault) {
	n := s.cur.Add(1) // 1-indexed match number
	last := len(s.stages) - 1
	for i, end := range s.cum {
		if i == last && s.openEnded {
			return true, s.stages[i].Faults
		}
		if n <= end {
			return true, s.stages[i].Faults
		}
	}
	return false, nil // past the last finite stage, not open-ended
}

// WithStages makes a rule inject different faults as its cumulative match count
// grows: stage 1 covers the first Stage.Times matches, stage 2 the next, and so
// on. A final stage with Times == 0 fires forever; otherwise the rule goes quiet
// once the last stage is exhausted (it falls through, like a spent Times).
//
// WithStages sets the rule's fire strategy: it is mutually exclusive with
// Times/Range/Probability/Sequence (last option wins) and takes precedence over
// StickyAttr. Panics if stages is empty, or if any non-final stage has
// Times <= 0 — the same fail-fast contract as Probability.
func WithStages(stages ...Stage) RuleOption {
	if err := validateStages(stages); err != nil {
		panic(err.Error())
	}
	sc := newStagedCounter(stages)
	return func(r *Rule) {
		r.staged = sc
	}
}
