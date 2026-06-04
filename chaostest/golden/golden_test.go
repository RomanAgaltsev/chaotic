package golden

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestEventsJSONLRoundTrip(t *testing.T) {
	events := []goldenEvent{
		{Fired: true, Rule: "a", Kind: int(engine.OpHTTPClient), Name: "/x", FaultKind: int(fault.KindLatency), LatencyNS: int64(5 * time.Millisecond)},
		{Fired: false, Rule: "a", Kind: int(engine.OpHTTPClient), Name: "/y", Reason: engine.ReasonCounter},
	}
	var buf bytes.Buffer
	if err := writeEvents(&buf, events); err != nil {
		t.Fatal(err)
	}
	got, err := readEvents(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Rule != "a" || got[0].FaultKind != int(fault.KindLatency) || got[1].Reason != engine.ReasonCounter {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestRecorderCapturesFiresSkipsAndLatency(t *testing.T) {
	rec := &recorder{}
	e := engine.New(engine.WithObserver(rec)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.Times(1),
		engine.WithFault(fault.Latency(7*time.Millisecond)),
	).Named("slow"))
	ctx := context.Background()
	act := e.Eval(ctx, engine.Op{Kind: engine.OpHTTPClient, Name: "/a"}) // fires
	_ = act.Before(ctx)                                                  // triggers FaultInjected (latency)
	_ = act.After(ctx)
	e.Eval(ctx, engine.Op{Kind: engine.OpHTTPClient, Name: "/b"}) // skipped (counter)

	got := rec.snapshot()
	if len(got) != 2 {
		t.Fatalf("events = %d, want 2 (%+v)", len(got), got)
	}
	if !got[0].Fired || got[0].FaultKind != int(fault.KindLatency) || got[0].LatencyNS != int64(7*time.Millisecond) {
		t.Fatalf("fired event = %+v, want latency 7ms", got[0])
	}
	if got[1].Fired || got[1].Reason != engine.ReasonCounter {
		t.Fatalf("skip event = %+v, want counter skip", got[1])
	}
}

// runWorkload evaluates a fixed op sequence against eng, exercising Before/After
// so latency faults and outcomes are realized, and returns nothing — the
// attached recorder captures the decisions.
func runWorkload(eng *engine.Engine, ops []engine.Op) {
	ctx := context.Background()
	for _, op := range ops {
		act := eng.Eval(ctx, op)
		_ = act.Before(ctx)
		_ = act.After(ctx)
	}
}

func TestReplayReproducesFireSequence(t *testing.T) {
	ops := []engine.Op{
		{Kind: engine.OpHTTPClient, Name: "/a"},
		{Kind: engine.OpHTTPClient, Name: "/b"},
		{Kind: engine.OpHTTPClient, Name: "/c"},
	}
	// Original engine: fire on the 1st and 3rd matches only.
	orig := &recorder{}
	engA := engine.New(engine.WithObserver(orig)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.Sequence([]bool{true, false, true}),
		engine.WithFault(fault.Error(errors.New("boom"))),
	).Named("flap"))
	runWorkload(engA, ops)
	recorded := orig.snapshot()

	// Replay: reconstruct rules, run the same workload, capture, compare.
	replayed := &recorder{}
	engB := engine.New(engine.WithObserver(replayed), engine.WithRuleSource(engine.NewRuleSet(buildReplayRules(recorded))))
	runWorkload(engB, ops)

	if diff := diffSequences(firedNames(recorded), firedNames(replayed.snapshot())); diff != "" {
		t.Fatalf("replay diverged: %s", diff)
	}
	if got := firedNames(replayed.snapshot()); len(got) != 2 {
		t.Fatalf("replay fired %d times, want 2 (%v)", len(got), got)
	}
}

func TestReplayReconstructsLatencyAndError(t *testing.T) {
	latRules := buildReplayRules([]goldenEvent{
		{Fired: true, Rule: "lat", Kind: int(engine.OpHTTPClient), FaultKind: int(fault.KindLatency), LatencyNS: int64(3 * time.Millisecond)},
	})
	if len(latRules) != 1 {
		t.Fatalf("rules = %d, want 1", len(latRules))
	}
	info := latRules[0].Info()
	if len(info.Faults) != 1 || info.Faults[0] != fault.KindLatency {
		t.Fatalf("latency rule faults = %v, want [KindLatency]", info.Faults)
	}
	errRules := buildReplayRules([]goldenEvent{
		{Fired: true, Rule: "err", Kind: int(engine.OpHTTPClient), FaultKind: int(fault.KindError)},
	})
	if got := errRules[0].Info().Faults[0]; got != fault.KindError {
		t.Fatalf("error rule fault = %v, want KindError (generic ErrReplay)", got)
	}
}

func TestWriteReadGoldenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.golden")
	events := []goldenEvent{{Fired: true, Rule: "a", Kind: int(engine.OpHTTPClient)}}
	if err := writeGolden(path, events); err != nil {
		t.Fatal(err)
	}
	got, err := readGolden(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Rule != "a" {
		t.Fatalf("round trip = %+v", got)
	}
}

func TestRecordReturnsObserverOption(t *testing.T) {
	// Record returns an engine.Option; building an engine with it must install a
	// recording observer (a fired rule is captured by the recorder it wires).
	opt := Record(t, "unused-no-update") // no -chaos-update-golden => cleanup won't write
	e := engine.New(opt).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(fault.Error(errors.New("x"))),
	).Named("r"))
	if got := e.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient}); got == engine.Pass {
		t.Fatal("expected a firing action, got Pass")
	}
}

func TestReplayOptionRebuildsRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seq.golden")
	if err := writeGolden(path, []goldenEvent{
		{Fired: true, Rule: "flap", Kind: int(engine.OpHTTPClient)},
		{Fired: false, Rule: "flap", Kind: int(engine.OpHTTPClient), Reason: engine.ReasonCounter},
	}); err != nil {
		t.Fatal(err)
	}
	rules := buildReplayRules(mustRead(t, path))
	e := engine.New(engine.WithRuleSource(engine.NewRuleSet(rules)))
	ctx := context.Background()
	if e.Eval(ctx, engine.Op{Kind: engine.OpHTTPClient}) == engine.Pass {
		t.Fatal("first eval should fire (mask[0]=true)")
	}
	if e.Eval(ctx, engine.Op{Kind: engine.OpHTTPClient}) != engine.Pass {
		t.Fatal("second eval should skip (mask[1]=false)")
	}
}

func mustRead(t *testing.T, path string) []goldenEvent {
	t.Helper()
	ev, err := readGolden(path)
	if err != nil {
		t.Fatal(err)
	}
	return ev
}
