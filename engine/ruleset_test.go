package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestNewRuleSetBacksEngineViaWithRuleSource(t *testing.T) {
	rs := NewRuleSet([]Rule{
		NewRule(MatchKind(OpHTTPClient), WithFault(fault.Error(errors.New("x")))).Named("r"),
	})
	e := New(WithRuleSource(rs))
	if !e.Enabled() {
		t.Fatal("engine with a rule source should be Enabled")
	}
	if a := e.Eval(context.Background(), Op{Kind: OpHTTPClient}); a == Pass {
		t.Fatal("expected the source's rule to fire")
	}
}

func TestReplaceRulesSwapsAtomically(t *testing.T) {
	e := New().AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Error(errors.New("old")))).Named("old"))
	// Replace with a brand-new set; the old rule must be gone.
	e.ReplaceRules(NewRuleSet([]Rule{
		NewRule(MatchKind(OpSQL), WithFault(fault.Error(errors.New("new")))).Named("new"),
	}))
	if a := e.Eval(context.Background(), Op{Kind: OpHTTPClient}); a != Pass {
		t.Fatal("old rule should no longer match after ReplaceRules")
	}
	if a := e.Eval(context.Background(), Op{Kind: OpSQL}); a == Pass {
		t.Fatal("new rule should match after ReplaceRules")
	}
}
