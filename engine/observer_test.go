package engine

import (
	"context"
	"testing"
)

type fakeObserver struct {
	fired   []string
	skipped []string
}

func (f *fakeObserver) RuleFired(named string, _ Op, _ Action) {
	f.fired = append(f.fired, named)
}

func (f *fakeObserver) RuleSkipped(named string, _ Op, _ string) {
	f.skipped = append(f.skipped, named)
}

func TestObserverInterfaceCompiles(t *testing.T) {
	var _ Observer = (*fakeObserver)(nil)
}

func TestKillSwitchTypeIsCallable(t *testing.T) {
	var ks KillSwitch = func(_ context.Context, _ Op) bool {
		return true
	}
	if !ks(context.Background(), Op{}) {
		t.Fatal("kill switch returned false")
	}
}

func TestSkipReasonsAreDistinctAndNonEmpty(t *testing.T) {
	reasons := []string{
		ReasonCounter,
		ReasonRateLimit,
		ReasonMaxConcurrent,
		ReasonFailureBudget,
		ReasonDisabled,
		ReasonKillSwitch,
	}
	seen := map[string]bool{}
	for _, r := range reasons {
		if r == "" {
			t.Fatal("empty reason constant")
		}
		if seen[r] {
			t.Fatalf("duplicate reason %q", r)
		}
		seen[r] = true
	}
}
