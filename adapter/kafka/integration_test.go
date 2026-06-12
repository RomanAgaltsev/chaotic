//go:build integration && !chaos_off

package kafka_test

import (
	"context"
	"errors"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"

	chaoskafka "github.com/RomanAgaltsev/chaotic/adapter/kafka"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestWrapperAgainstRealKafka(t *testing.T) {
	ctx := context.Background()

	ctr, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.6.1")
	if err != nil {
		t.Skipf("cannot start Kafka container (Docker unavailable?): %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	brokers, err := ctr.Brokers(ctx)
	if err != nil {
		t.Fatalf("Brokers: %v", err)
	}

	const topic = "chaos-test"
	w := &kafkago.Writer{Addr: kafkago.TCP(brokers...), Topic: topic, AllowAutoTopicCreation: true}
	t.Cleanup(func() { _ = w.Close() })

	eng := engine.New()
	cw := chaoskafka.WrapWriter(w, eng)

	wctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Happy path: write a message.
	if err := cw.WriteMessages(wctx, kafkago.Message{Value: []byte("hello")}); err != nil {
		t.Fatalf("WriteMessages: %v", err)
	}

	// Read it back through a wrapped reader (promoted Close + faulted ReadMessage).
	r := kafkago.NewReader(kafkago.ReaderConfig{Brokers: brokers, Topic: topic, GroupID: "g"})
	t.Cleanup(func() { _ = r.Close() })
	cr := chaoskafka.WrapReader(r, eng)
	rctx, rcancel := context.WithTimeout(ctx, 15*time.Second)
	defer rcancel()
	msg, err := cr.ReadMessage(rctx)
	if err != nil || string(msg.Value) != "hello" {
		t.Fatalf("ReadMessage = %q err=%v; want \"hello\"", msg.Value, err)
	}

	// Fault path: a rule that fails the write surfaces the supplied error.
	sentinel := errors.New("chaos")
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpKafka),
		engine.MatchName(topic),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("write-fail"))

	if err := cw.WriteMessages(wctx, kafkago.Message{Value: []byte("hello")}); !errors.Is(err, sentinel) {
		t.Fatalf("WriteMessages err = %v, want sentinel", err)
	}
}
