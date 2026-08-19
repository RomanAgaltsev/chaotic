package engine

import (
	"errors"
	"strings"
	"testing"
)

// highSpec is a spec LintSpecs rates SeverityHigh: a terminal fault with no
// kind and no name scope matches every operation.
func highSpec() RuleSpec {
	return RuleSpec{
		Name:   "wipeout",
		Faults: []FaultSpec{{Type: "panic", Message: "boom"}},
	}
}

// warnSpec is a spec LintSpecs rates SeverityWarn only: a wildcard name glob
// with a non-terminal fault.
func warnSpec() RuleSpec {
	return RuleSpec{
		Name:     "slowish",
		Kinds:    []string{"sql"},
		NameGlob: "*",
		Faults:   []FaultSpec{{Type: "latency", Duration: "10ms"}},
	}
}

func TestLintGateOffSkipsLinting(t *testing.T) {
	rep, err := LintGate(LintOff, []RuleSpec{highSpec()})
	if err != nil {
		t.Fatalf("LintOff returned error: %v", err)
	}
	if len(rep.Findings) != 0 {
		t.Fatalf("LintOff produced %d findings, want 0", len(rep.Findings))
	}
}

func TestLintGateWarnReportsButAllows(t *testing.T) {
	rep, err := LintGate(LintWarn, []RuleSpec{highSpec()})
	if err != nil {
		t.Fatalf("LintWarn returned error: %v", err)
	}
	if len(rep.Findings) == 0 {
		t.Fatal("LintWarn produced no findings, want at least one")
	}
}

func TestLintGateRejectFailsOnHigh(t *testing.T) {
	rep, err := LintGate(LintReject, []RuleSpec{highSpec()})
	if !errors.Is(err, ErrLintRejected) {
		t.Fatalf("LintReject error = %v, want ErrLintRejected", err)
	}
	if !strings.Contains(err.Error(), "wipeout") {
		t.Errorf("error %q does not name the offending rule", err)
	}
	if len(rep.Findings) == 0 {
		t.Error("LintReject returned no findings; the report must still be reportable")
	}
}

func TestLintGateRejectAllowsWarnOnly(t *testing.T) {
	rep, err := LintGate(LintReject, []RuleSpec{warnSpec()})
	if err != nil {
		t.Fatalf("LintReject rejected a warn-only spec: %v", err)
	}
	if len(rep.Findings) == 0 {
		t.Fatal("expected warn findings to still be reported under LintReject")
	}
}

func TestLintModeString(t *testing.T) {
	for mode, want := range map[LintMode]string{
		LintOff: "off", LintWarn: "warn", LintReject: "reject", LintMode(99): "unknown",
	} {
		if got := mode.String(); got != want {
			t.Errorf("LintMode(%d).String() = %q, want %q", int(mode), got, want)
		}
	}
}

func TestReportSummary(t *testing.T) {
	if got := (Report{}).Summary(); got != "no findings" {
		t.Errorf("empty Summary() = %q, want %q", got, "no findings")
	}
	rep := Report{Findings: []Finding{
		{Severity: SeverityHigh, Rule: "r1", Message: "bad"},
		{Severity: SeverityWarn, Rule: "r2", Message: "meh"},
	}}
	want := "[high] r1: bad; [warn] r2: meh"
	if got := rep.Summary(); got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}
}
