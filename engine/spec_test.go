package engine

import (
	"context"
	"strings"
	"testing"
)

func TestBuildRuleConstructsMatchingRule(t *testing.T) {
	spec := RuleSpec{
		Name:     "transient",
		Kinds:    []string{"http_client"},
		NameGlob: "/users/*",
		Counter:  CounterSpec{Type: "times", N: 2},
		Faults:   []FaultSpec{{Type: "error", Message: "boom"}},
	}
	r, err := BuildRule(spec)
	if err != nil {
		t.Error(err)
	}
	if r.Name() != "transient" {
		t.Fatalf("name = %q, want transient", r.Name())
	}
	e := New().AddRule(r)
	if a := e.Eval(context.Background(), Op{Kind: OpHTTPClient, Name: "/users/123"}); a == Pass {
		t.Fatal("expected rule to fire for http_client /users/123")
	}
	if a := e.Eval(context.Background(), Op{Kind: OpSQL, Name: "/users/123"}); a != Pass {
		t.Fatal("expected no match for sql")
	}
}

func TestBuildRuleRejectsUnknownKind(t *testing.T) {
	if _, err := BuildRule(RuleSpec{Kinds: []string{"nope"}}); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestBuildRuleRejectsBadDuration(t *testing.T) {
	_, err := BuildRule(RuleSpec{Faults: []FaultSpec{{Type: "latency", Duration: "abc"}}})
	if err == nil {
		t.Fatal("expected error for unparseable duration")
	}
}

func TestBuildRuleRejectsUnknownFault(t *testing.T) {
	if _, err := BuildRule(RuleSpec{Faults: []FaultSpec{{Type: "explode"}}}); err == nil {
		t.Fatal("expected error for unknown fault type")
	}
}

func TestBuildRuleRejectsOutOfRangeProbability(t *testing.T) {
	_, err := BuildRule(RuleSpec{
		Counter: CounterSpec{
			Type: "probability",
			P:    2.0,
		},
	})
	if err == nil {
		t.Fatal("expected error for p=2.0, got nil (did it panic instead?)")
	}
}

func TestBuildRuleRejectsTooLargeLatency(t *testing.T) {
	_, err := BuildRule(RuleSpec{
		Faults: []FaultSpec{{Type: "latency", Duration: "10m"}},
	})
	if err == nil {
		t.Fatal("expected error for latency > maxFaultLatency, got nil")
	}
}

func TestBuildRuleReportsAllErrors(t *testing.T) {
	_, err := BuildRule(RuleSpec{
		Kinds:   []string{"not_a_kind"},
		Counter: CounterSpec{Type: "probability", P: -1},
		Faults:  []FaultSpec{{Type: "latency", Duration: "nope"}},
	})
	if err == nil {
		t.Fatal("expected aggregated error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"not_a_kind", "[0,1]", "nope"} {
		if !strings.Contains(msg, want) {
			t.Errorf("aggregated error missing %q; got: %v", want, msg)
		}
	}
}
