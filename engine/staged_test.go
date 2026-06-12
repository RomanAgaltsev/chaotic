package engine

import (
	"context"
	"sync"
	"testing"

	"github.com/ag4r/chaotic/fault"
)

func newTestCtx() context.Context { return context.Background() }

func TestStagedCounterWindowing(t *testing.T) {
	lat := fault.Latency(0)
	errF := fault.Error(errStagedTest)
	drop := fault.ConnDrop()
	sc := newStagedCounter([]Stage{
		{Times: 2, Faults: []fault.Fault{lat}},
		{Times: 1, Faults: []fault.Fault{errF}},
		{Times: 0, Faults: []fault.Fault{drop}}, // forever
	})

	want := [][]fault.Fault{
		{lat}, {lat}, // matches 1-2
		{errF},                 // match 3
		{drop}, {drop}, {drop}, // matches 4-6 (forever)
	}
	for i, w := range want {
		fire, got := sc.fire()
		if !fire {
			t.Fatalf("match %d: fire = false, want true", i+1)
		}
		if len(got) != len(w) || (len(w) == 1 && got[0] != w[0]) {
			t.Fatalf("match %d: faults = %v, want %v", i+1, got, w)
		}
	}
}

func TestStagedCounterFiniteTailGoesQuiet(t *testing.T) {
	lat := fault.Latency(0)
	errF := fault.Error(errStagedTest)
	sc := newStagedCounter([]Stage{
		{Times: 2, Faults: []fault.Fault{lat}},
		{Times: 3, Faults: []fault.Fault{errF}},
	})
	for i := range 5 { // matches 1-5 all fire
		if fire, _ := sc.fire(); !fire {
			t.Fatalf("match %d should fire", i+1)
		}
	}
	if fire, faults := sc.fire(); fire || faults != nil { // match 6
		t.Fatalf("match 6: fire=%v faults=%v, want false nil", fire, faults)
	}
}

func TestStagedCounterEmptyFaultsStageFires(t *testing.T) {
	sc := newStagedCounter([]Stage{
		{Times: 1, Faults: nil},                             // pass-through window
		{Times: 0, Faults: []fault.Fault{fault.Latency(0)}}, // then forever
	})
	if fire, faults := sc.fire(); !fire || faults != nil {
		t.Fatalf("match 1: fire=%v faults=%v, want true nil", fire, faults)
	}
	if fire, faults := sc.fire(); !fire || len(faults) != 1 {
		t.Fatalf("match 2: fire=%v faults=%v, want true [latency]", fire, faults)
	}
}

func TestStagedCounterConcurrentFirePartitions(t *testing.T) {
	lat := fault.Latency(0)
	errF := fault.Error(errStagedTest)
	// 100 in stage A, 100 in stage B; after 200, quiet.
	sc := newStagedCounter([]Stage{
		{Times: 100, Faults: []fault.Fault{lat}},
		{Times: 100, Faults: []fault.Fault{errF}},
	})
	var mu sync.Mutex
	counts := map[fault.Fault]int{}
	var wg sync.WaitGroup
	for range 250 { // 250 calls: 100 A, 100 B, 50 quiet
		wg.Add(1)
		go func() {
			defer wg.Done()
			fire, faults := sc.fire()
			mu.Lock()
			defer mu.Unlock()
			if !fire {
				counts[nil]++
				return
			}
			counts[faults[0]]++
		}()
	}
	wg.Wait()
	if counts[lat] != 100 || counts[errF] != 100 || counts[nil] != 50 {
		t.Fatalf("partition wrong: lat=%d err=%d quiet=%d, want 100/100/50",
			counts[lat], counts[errF], counts[nil])
	}
}

var errStagedTest = stagedTestErr("staged boom")

type stagedTestErr string

func (e stagedTestErr) Error() string { return string(e) }

func TestWithStagesSetsStagedField(t *testing.T) {
	r := NewRule(
		MatchKind(OpHTTPClient),
		WithStages(
			Stage{Times: 1, Faults: []fault.Fault{fault.Latency(0)}},
			Stage{Times: 0, Faults: []fault.Fault{fault.ConnDrop()}},
		),
	)
	if r.staged == nil {
		t.Fatal("WithStages should set r.staged")
	}
	if !r.staged.openEnded {
		t.Fatal("final Times==0 should mark openEnded")
	}
}

func TestWithStagesPanics(t *testing.T) {
	tests := []struct {
		name   string
		stages []Stage
	}{
		{"empty", nil},
		{"nonfinal_zero", []Stage{{Times: 0, Faults: nil}, {Times: 1, Faults: nil}}},
		{"nonfinal_negative", []Stage{{Times: -1, Faults: nil}, {Times: 1, Faults: nil}}},
		{"final_negative", []Stage{{Times: 1, Faults: nil}, {Times: -1, Faults: nil}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			WithStages(tt.stages...)
		})
	}
}

func TestEvalStagedSelectsActiveFaults(t *testing.T) {
	eng := New().AddRule(NewRule(
		MatchKind(OpHTTPClient),
		WithStages(
			Stage{Times: 2, Faults: []fault.Fault{fault.Error(errStagedTest)}},
			Stage{Times: 0, Faults: nil}, // then quiet-but-present forever (no fault)
		),
	).Named("staged"))

	op := Op{Kind: OpHTTPClient, Name: "/x"}
	ctx := newTestCtx()
	got := make([]error, 3)
	for i := range got {
		got[i] = eng.Eval(ctx, op).Before(ctx)
	}
	if got[0] == nil || got[1] == nil {
		t.Fatalf("matches 1-2 should inject the error, got %v", got)
	}
	if got[2] != nil {
		t.Fatalf("match 3 (empty-faults stage) should inject nothing, got %v", got[2])
	}
	if eng.Hits("staged") != 3 {
		t.Fatalf("hits = %d, want 3 (every match fires)", eng.Hits("staged"))
	}
}

func TestEvalStagedReleasesMaxConcurrentSlot(t *testing.T) {
	eng := New(WithMaxConcurrent(1)).AddRule(NewRule(
		MatchKind(OpHTTPClient),
		WithStages(Stage{Times: 0, Faults: []fault.Fault{fault.Error(errStagedTest)}}),
	).Named("staged"))

	op := Op{Kind: OpHTTPClient, Name: "/x"}
	ctx := newTestCtx()
	for i := range 3 { // if After didn't release, call 2 would be starved
		act := eng.Eval(ctx, op)
		if err := act.Before(ctx); err == nil {
			t.Fatalf("call %d: expected injected error", i+1)
		}
		_ = act.After(ctx)
	}
}
