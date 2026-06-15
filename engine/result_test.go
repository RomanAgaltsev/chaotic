package engine_test

import (
	"context"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// appendMutator is a test fault that is a no-op in Before and appends suffix to
// a string result. It implements both fault.Fault (Apply) and the engine's
// structurally-detected MutateResult.
type appendMutator struct{ suffix string }

func (appendMutator) Apply(context.Context) error   { return nil }
func (m appendMutator) MutateResult(result any) any { s, _ := result.(string); return s + m.suffix }

// panicOnApplyMutator proves result mutators are excluded from the Before chain:
// its Apply panics, so a passing Before means Apply was never called.
type panicOnApplyMutator struct{}

func (panicOnApplyMutator) Apply(context.Context) error {
	panic("Apply must not run for a result mutator")
}
func (panicOnApplyMutator) MutateResult(result any) any { return result }

func TestResultReporterAppliesMutatorsInOrder(t *testing.T) {
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(appendMutator{"-a"}),
		engine.WithFault(appendMutator{"-b"}),
	))
	act := e.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient})
	if err := act.Before(context.Background()); err != nil {
		t.Fatalf("Before: %v", err)
	}
	rr, ok := act.(engine.ResultReporter)
	if !ok {
		t.Fatal("firing action does not implement ResultReporter")
	}
	if got := rr.Result(context.Background(), "x"); got != "x-a-b" {
		t.Fatalf("Result = %q, want %q", got, "x-a-b")
	}
}

func TestPassIsNotResultReporter(t *testing.T) {
	if _, ok := engine.Pass.(engine.ResultReporter); ok {
		t.Fatal("Pass must not implement ResultReporter")
	}
}

func TestResultMutatorExcludedFromBeforeChain(t *testing.T) {
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(panicOnApplyMutator{}),
	))
	act := e.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient})
	if err := act.Before(context.Background()); err != nil {
		t.Fatalf("Before should be nil (mutator excluded), got %v", err)
	}
}
