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
	kinds := []Kind{OpHTTPClient, OpHTTPServer, OpSQL, OpGRPCClient, OpGRPCServer}
	seen := map[Kind]bool{}
	for _, k := range kinds {
		if seen[k] {
			t.Fatalf("duplicate kind value %d", k)
		}
		seen[k] = true
	}
}
