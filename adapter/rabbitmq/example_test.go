//go:build !chaos_off

package rabbitmq

import (
	"context"
	"errors"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func ExampleWrapChannel() {
	// Fail the first publish, then recover.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("broker down"))),
	).Named("amqp-flap"))

	// In production you wrap a live channel: ch := chaosrabbitmq.WrapChannel(real, eng).
	// Here a fake stands in for the broker so the example is hermetic.
	c := &Channel{ch: &fakeChannel{}, eng: eng}

	publish := func() error {
		return c.PublishWithContext(context.Background(), "events", "order.created", false, false, amqp.Publishing{Body: []byte("order")})
	}

	fmt.Println("attempt 1:", publish())
	fmt.Println("attempt 2:", publish())
	// Output:
	// attempt 1: broker down
	// attempt 2: <nil>
}
