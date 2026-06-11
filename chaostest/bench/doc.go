// Package bench runs one benchmark body across a series of chaos profiles, so you
// can see how a metric (e.g. ns/op, allocs/op) moves as the chaos profile
// changes — "how does throughput degrade as I add latency?".
//
// Declare profiles as a slice (order is stable, so benchstat compares rows
// predictably) and call Run:
//
//	func BenchmarkCheckout(b *testing.B) {
//	    eng := engine.New()
//	    profiles := []bench.Profile{
//	        {Name: "baseline", Apply: nil},
//	        {Name: "latency-5ms", Apply: func(e *engine.Engine) {
//	            e.AddRule(engine.NewRule(engine.MatchKind(engine.OpHTTPClient),
//	                engine.WithFault(fault.Latency(5*time.Millisecond))).Named("lat"))
//	        }},
//	    }
//	    bench.Run(b, eng, profiles, func(sub *testing.B) {
//	        for i := 0; i < sub.N; i++ {
//	            // ... exercise the system under test against eng ...
//	        }
//	    })
//	}
//
// Run reports ns/op and allocs/op per profile. Pipe `go test -bench .` output
// into benchstat to compare. (Per-op latency percentiles are a planned follow-on.)
package bench
