package engine

import (
	"errors"
	"strings"
	"testing"

	"github.com/ag4r/chaotic/fault"
)

// hasSeverity reports whether rep contains at least one finding at sev.
func hasSeverity(rep Report, sev Severity) bool {
	for _, f := range rep.Findings {
		if f.Severity == sev {
			return true
		}
	}
	return false
}

func hasFinding(rep Report, sev Severity, substr string) bool {
	for _, f := range rep.Findings {
		if f.Severity == sev && strings.Contains(f.Message, substr) {
			return true
		}
	}
	return false
}

func TestLintFlagsUnconstrainedPanic(t *testing.T) {
	rules := []Rule{
		NewRule(WithFault(fault.Panic("boom"))).Named("nuke"),
	}
	rep := Lint(rules)
	if rep.OK() {
		t.Fatal("expected a HIGH finding for an unconstrained panic rule")
	}
	if !hasSeverity(rep, SeverityHigh) {
		t.Fatalf("want a HIGH finding, got %+v", rep.Findings)
	}
}

func TestLintFlagsUnconstrainedConnDrop(t *testing.T) {
	rules := []Rule{
		NewRule(WithFault(fault.ConnDrop())).Named("drop"),
	}
	rep := Lint(rules)
	if !hasSeverity(rep, SeverityHigh) {
		t.Fatalf("want a HIGH finding for an unconstrained conn-drop rule, got %+v", rep.Findings)
	}
}

func TestLintPassesWellScopedRule(t *testing.T) {
	rules := []Rule{
		NewRule(MatchKind(OpSQL), Times(3), WithFault(fault.Error(errors.New("x")))).Named("ok"),
	}
	rep := Lint(rules)
	if !rep.OK() {
		t.Fatalf("expected no HIGH findings for a well-scoped rule, got %+v", rep.Findings)
	}
	if len(rep.Findings) != 0 {
		t.Fatalf("expected an empty report for a well-scoped rule, got %+v", rep.Findings)
	}
}

func TestLintSpecsFlagsWildcardCertainProbability(t *testing.T) {
	specs := []RuleSpec{{
		Name:     "broad",
		NameGlob: "*",
		Counter:  CounterSpec{Type: "probability", P: 1.0},
		Faults:   []FaultSpec{{Type: "error", Message: "x"}},
	}}
	rep := LintSpecs(specs)
	if len(rep.Findings) == 0 {
		t.Fatal("expected a finding for a wildcard glob with probability 1.0")
	}
}

func TestLintSpecsFlagsExcessiveLatency(t *testing.T) {
	specs := []RuleSpec{{
		Name:    "slow",
		Kinds:   []string{"sql"},
		Counter: CounterSpec{Type: "times", N: 1},
		Faults:  []FaultSpec{{Type: "latency", Duration: "30s"}},
	}}
	rep := LintSpecs(specs)
	if len(rep.Findings) == 0 {
		t.Fatal("expected a finding for latency above the ceiling")
	}
}

func TestLintSpecsFlagsOverlap(t *testing.T) {
	specs := []RuleSpec{
		{
			Name:     "a",
			Kinds:    []string{"http_client"},
			NameGlob: "GET",
			Faults:   []FaultSpec{{Type: "latency", Duration: "1ms"}},
		},
		{
			Name:     "b",
			Kinds:    []string{"http_client"},
			NameGlob: "GET",
			Faults:   []FaultSpec{{Type: "latency", Duration: "1ms"}},
		},
	}
	rep := LintSpecs(specs)
	if len(rep.Findings) == 0 {
		t.Fatal("expected an overlap finding for two specs with identical kind+glob")
	}
}

func TestLintSpecsPassesWellScopedSpec(t *testing.T) {
	specs := []RuleSpec{{
		Name:     "ok",
		Kinds:    []string{"sql"},
		NameGlob: "SELECT",
		Counter:  CounterSpec{Type: "times", N: 5},
		Faults:   []FaultSpec{{Type: "latency", Duration: "50ms"}},
	}}
	rep := LintSpecs(specs)
	if len(rep.Findings) != 0 {
		t.Fatalf("expected an empty report for a well-scoped spec, got %+v", rep.Findings)
	}
}

func TestLintSpecsOpenEndedTerminalStageWarns(t *testing.T) {
	rep := LintSpecs([]RuleSpec{{
		Name:  "degrade",
		Kinds: []string{"http_client"},
		Stages: []StageSpec{
			{Times: 2, Faults: []FaultSpec{{Type: "latency", Duration: "10ms"}}},
			{Times: 0, Faults: []FaultSpec{{Type: "conn_drop"}}},
		},
	}})
	if !hasFinding(rep, SeverityWarn, "fails permanently") {
		t.Fatalf("expected open-ended-terminal warning, got %+v", rep.Findings)
	}
}

func TestLintSpecsStageFaultsFlowThroughChecks(t *testing.T) {
	// Broad scope (no kind, no glob) + a panic stage fault => High, same as a flat rule.
	rep := LintSpecs([]RuleSpec{{
		Name:   "boom",
		Stages: []StageSpec{{Times: 0, Faults: []FaultSpec{{Type: "panic", Value: "x"}}}},
	}})
	if rep.OK() {
		t.Fatal("broad-scope panic stage fault should produce a High finding")
	}
}
