package engine

import (
	"context"
	"testing"
)

func TestPassActionIsNoOp(t *testing.T) {
	if err := Pass.Before(context.Background()); err != nil {
		t.Fatalf("Pass.Before returned %v, want nil", err)
	}
	if err := Pass.After(context.Background()); err != nil {
		t.Fatalf("Pass.After returned %v, want nil", err)
	}
}

func TestKindsAreDistinct(t *testing.T) {
	kinds := []Kind{OpHTTPClient, OpHTTPServer, OpSQL, OpGRPCClient, OpGRPCServer, OpExplicit, OpPGX}
	seen := map[Kind]bool{}
	for _, k := range kinds {
		if seen[k] {
			t.Fatalf("duplicate kind value %d", k)
		}
		seen[k] = true
	}
}

func TestOpExplicitIsSixthKind(t *testing.T) {
	if OpExplicit != OpGRPCServer+1 {
		t.Fatalf("OpExplicit = %d, want %d (appended after OpGRPCServer)", OpExplicit, OpGRPCServer+1)
	}
}

func TestBuildRuleAcceptsExplicitKind(t *testing.T) {
	r, err := BuildRule(RuleSpec{Kinds: []string{"explicit"}})
	if err != nil {
		t.Fatalf("BuildRule with explicit kind: %v", err)
	}
	if !r.matches(context.Background(), Op{Kind: OpExplicit, Name: "p"}) {
		t.Fatal("rule did not match an OpExplicit op")
	}
}
