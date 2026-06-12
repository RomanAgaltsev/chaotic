package scenarios_test

import (
	"context"
	"fmt"

	"github.com/RomanAgaltsev/chaotic/chaostest/scenarios"
	"github.com/RomanAgaltsev/chaotic/engine"
)

func ExampleDatabaseOutageCascade() {
	eng := engine.New()
	scenarios.DatabaseOutageCascade(eng, scenarios.WithCount(1))

	// The first SQL call hits the outage.
	act := eng.Eval(context.Background(), engine.Op{Kind: engine.OpSQL, Name: "SELECT"})
	fmt.Println(act.Before(context.Background()) != nil)
	// Output: true
}

func ExampleAWSRegionFailover() {
	eng := engine.New()
	scenarios.AWSRegionFailover(eng, scenarios.WithCount(1))

	// The first AWS call hits the region outage.
	act := eng.Eval(context.Background(), engine.Op{Kind: engine.OpAWS, Name: "s3.GetObject"})
	fmt.Println(act.Before(context.Background()) != nil)
	// Output: true
}
