package scenarios_test

import (
	"context"
	"fmt"

	"github.com/ag4r/chaotic/chaostest/scenarios"
	"github.com/ag4r/chaotic/engine"
)

func ExampleDatabaseOutageCascade() {
	eng := engine.New()
	scenarios.DatabaseOutageCascade(eng, scenarios.WithCount(1))

	// The first SQL call hits the outage.
	act := eng.Eval(context.Background(), engine.Op{Kind: engine.OpSQL, Name: "SELECT"})
	fmt.Println(act.Before(context.Background()) != nil)
	// Output: true
}
