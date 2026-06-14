package fault_test

import (
	"context"
	"testing"

	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestResponseMutateApplyIsNoOp(t *testing.T) {
	f := fault.ResponseMutate(func(v any) any { return "changed" })
	if err := f.Apply(context.Background()); err != nil {
		t.Fatalf("Apply = %v, want nil", err)
	}
}

func TestResponseMutateRunsFn(t *testing.T) {
	f := fault.ResponseMutate(func(v any) any { return v.(string) + "!" })
	m, ok := f.(interface{ MutateResult(any) any })
	if !ok {
		t.Fatal("ResponseMutate fault does not implement MutateResult")
	}
	if got := m.MutateResult("hi"); got != "hi!" {
		t.Fatalf("MutateResult = %v, want %q", got, "hi!")
	}
}

func TestResponseMutateKind(t *testing.T) {
	f := fault.ResponseMutate(func(v any) any { return v })
	if got := fault.KindOf(f); got != fault.KindResponseMutate {
		t.Fatalf("KindOf = %v, want KindResponseMutate", got)
	}
}

func TestResponseMutateNilFnPassesThrough(t *testing.T) {
	f := fault.ResponseMutate(nil)
	m := f.(interface{ MutateResult(any) any })
	if got := m.MutateResult("x"); got != "x" {
		t.Fatalf("nil fn should pass result through, got %v", got)
	}
}
