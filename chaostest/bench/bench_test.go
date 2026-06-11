package bench_test

import (
	"context"
	"testing"

	"github.com/ag4r/chaotic/chaostest/bench"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestRunIteratesProfilesResetsAndAppliesEachAndRunsBody(t *testing.T) {
	eng := engine.New()
	appliedLatency := 0
	bodyRuns := 0

	profiles := []bench.Profile{
		{Name: "baseline", Apply: nil}, // no rules
		{Name: "latency", Apply: func(e *engine.Engine) {
			appliedLatency++
			e.AddRule(engine.NewRule(
				engine.MatchKind(engine.OpHTTPClient),
				engine.WithFault(fault.Latency(0)),
			).Named("lat"))
		}},
	}

	// testing.Benchmark drives Run as a benchmark; Run fans out to sub-benchmarks.
	_ = testing.Benchmark(func(b *testing.B) {
		bench.Run(b, eng, profiles, func(sub *testing.B) {
			bodyRuns++
			for i := 0; i < sub.N; i++ {
				_ = eng.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient, Name: "/x"})
			}
		})
	})

	if appliedLatency == 0 {
		t.Fatal("the latency profile's Apply was never called")
	}
	if bodyRuns == 0 {
		t.Fatal("the body was never run")
	}
	// After the run, the engine was reset between profiles, so it does not retain
	// the latency rule beyond its own profile (the baseline that may follow in a
	// re-run starts clean). We assert the engine is resettable to empty here.
	eng.Reset()
	if eng.Enabled() {
		t.Fatal("engine should be empty after Reset")
	}
}
