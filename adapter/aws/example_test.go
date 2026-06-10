//go:build !chaos_off

package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/smithy-go/middleware"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func ExampleAppendChaosMiddleware() {
	// In production:
	//   cfg, _ := config.LoadDefaultConfig(ctx)
	//   chaosaws.AppendChaosMiddleware(&cfg, eng)
	//   ddb := dynamodb.NewFromConfig(cfg)
	// Here we drive the middleware directly so the example is hermetic.

	// Fail the first request, then recover.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpAWS),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("ThrottlingException"))),
	).Named("aws-flap"))

	mw := chaosMiddleware{eng: eng}
	next := middleware.FinalizeHandlerFunc(func(ctx context.Context, in middleware.FinalizeInput) (middleware.FinalizeOutput, middleware.Metadata, error) {
		return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
	})
	send := func() error {
		_, _, err := mw.HandleFinalize(context.Background(), middleware.FinalizeInput{}, next)
		return err
	}

	fmt.Println("attempt 1:", send())
	fmt.Println("attempt 2:", send())
	// Output:
	// attempt 1: ThrottlingException
	// attempt 2: <nil>
}
