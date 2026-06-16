//go:build !chaos_off

package kafka

import (
	"context"
	"errors"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func ExampleWrapWriter() {
	// Fail the first write, then recover.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpKafka),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("broker down"))),
	).Named("kafka-flap"))

	// In production you wrap a live writer: cw := chaoskafka.WrapWriter(real, eng).
	// Here a fake stands in for the broker so the example is hermetic.
	w := &Writer{w: &fakeWriter{}, eng: eng, topic: "events"}

	write := func() error {
		return w.WriteMessages(context.Background(), kafkago.Message{Value: []byte("order")})
	}

	fmt.Println("attempt 1:", write())
	fmt.Println("attempt 2:", write())
	// Output:
	// attempt 1: broker down
	// attempt 2: <nil>
}
