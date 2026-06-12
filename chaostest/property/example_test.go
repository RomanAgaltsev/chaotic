package property_test

import (
	"fmt"
	"math/rand/v2"

	"github.com/RomanAgaltsev/chaotic/chaostest/property"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func ExampleTest() {
	gens := []property.RuleGen{
		func(r *rand.Rand) engine.Rule {
			return engine.NewRule(
				engine.MatchKind(engine.OpHTTPClient),
				engine.Probability(r.Float64(), int64(r.Uint64())),
				engine.WithFault(fault.ConnDrop()),
			).Named("net")
		},
	}
	// A body that always holds; in a real test you would exercise your system.
	body := func(*engine.Engine) error { return nil }

	// property.Test takes a testing.TB; here we just show the wiring compiles.
	_ = gens
	_ = body
	fmt.Println("configured")
	// Output: configured
}
