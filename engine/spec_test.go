package engine

import (
	"context"
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
