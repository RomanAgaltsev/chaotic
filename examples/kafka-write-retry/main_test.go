package main

import (
	"context"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"

	chaoskafka "github.com/RomanAgaltsev/chaotic/adapter/kafka"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestWriteRetrySurvivesOutage(t *testing.T) {
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

	const topic = "retry-demo"
	w := &kafkago.Writer{Addr: kafkago.TCP(brokers...), Topic: topic, AllowAutoTopicCreation: true}
	t.Cleanup(func() { _ = w.Close() })

	eng := engine.New()
	cw := chaoskafka.WrapWriter(w, eng)

	// Drop the first two writes: a transient outage. ConnDrop returns
	// io.ErrUnexpectedEOF but never touches the real writer, so the retry's next
	// write lands on the still-open writer.
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpKafka),
		engine.MatchName(topic),
		engine.Times(2),
		engine.WithFault(fault.ConnDrop()),
	).Named("outage"))

	wctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := WriteWithRetry(wctx, cw, 5, kafkago.Message{Value: []byte("ok")}); err != nil {
		t.Fatalf("WriteWithRetry failed despite retries: %v", err)
	}
	if got := eng.Hits("outage"); got != 2 {
		t.Fatalf("outage fired %d times, want 2", got)
	}
}
