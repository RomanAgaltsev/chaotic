package chaostest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ag4r/chaotic/chaostest"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestRecorderCapturesFiresAndSkips(t *testing.T) {
	rec := chaostest.NewRecorder()
	e := chaostest.New(t, engine.WithObserver(rec)).
		AddRule(engine.NewRule(
			engine.MatchKind(engine.OpHTTPClient),
			engine.Times(1),
			engine.WithFault(fault.Error(errors.New("x"))),
		).Named("once"))

	e.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient, Name: "/a"}) // fires
	e.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient, Name: "/b"}) // skipped (counter)

	if got := len(rec.Fired("once")); got != 1 {
		t.Fatalf("Fired = %d, want 1", got)
	}
	skipped := rec.Skipped("once")
	if len(skipped) != 1 || skipped[0].Reason != engine.ReasonCounter {
		t.Fatalf("Skipped = %+v, want one with reason %q", skipped, engine.ReasonCounter)
	}
	if rec.Fired("once")[0].Op.Name != "/a" {
		t.Fatalf("recorded op name = %q, want /a", rec.Fired("once")[0].Op.Name)
	}
}
