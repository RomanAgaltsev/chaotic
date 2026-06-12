package bench_test

import (
	"fmt"

	"github.com/RomanAgaltsev/chaotic/chaostest/bench"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func ExampleRun() {
	profiles := []bench.Profile{
		{Name: "baseline", Apply: nil},
		{Name: "conn-drops", Apply: func(e *engine.Engine) {
			e.AddRule(engine.NewRule(
				engine.MatchKind(engine.OpHTTPClient),
				engine.WithFault(fault.ConnDrop()),
			).Named("drop"))
		}},
	}
	// bench.Run takes a *testing.B; here we just show the profiles are declared
	// in a stable order for benchstat.
	for _, p := range profiles {
		fmt.Println(p.Name)
	}
	// Output:
	// baseline
	// conn-drops
}
