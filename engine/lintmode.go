package engine

import (
	"errors"
	"fmt"
)

// LintMode selects what a rule source does with LintSpecs findings.
type LintMode int

// Lint modes, in ascending order of strictness.
const (
	// LintOff installs rules without linting them. It is the zero value, so a
	// source given no lint option behaves exactly as it did before lint wiring
	// existed, and pays nothing for the feature.
	LintOff LintMode = iota
	// LintWarn reports findings and installs the rules anyway.
	LintWarn
	// LintReject reports findings and fails the load when any finding is
	// SeverityHigh, matching Report.OK.
	LintReject
)

// String returns the lowercase mode name.
func (m LintMode) String() string {
	switch m {
	case LintOff:
		return "off"
	case LintWarn:
		return "warn"
	case LintReject:
		return "reject"
	}
	return "unknown"
}

// ErrLintRejected is the sentinel wrapped by every LintGate rejection.
var ErrLintRejected = errors.New("chaotic: rules rejected by lint")

// LintGate runs LintSpecs under mode and reports what the caller should do. It
// is the shared implementation behind every rule source's WithLint option, so
// the three modes mean the same thing in every source.
//
// LintOff skips the lint pass entirely and returns a zero Report. LintWarn
// returns the findings for the caller to report through whatever sink suits it.
// LintReject additionally returns an error wrapping ErrLintRejected when the
// report is not OK.
//
// The returned Report is always safe to report, in every mode and alongside a
// non-nil error — a rejecting caller still wants to tell the operator why.
func LintGate(mode LintMode, specs []RuleSpec) (Report, error) {
	if mode == LintOff {
		return Report{}, nil
	}
	rep := LintSpecs(specs)
	if mode == LintReject && !rep.OK() {
		return rep, fmt.Errorf("%w: %s", ErrLintRejected, rep.Summary())
	}
	return rep, nil
}
