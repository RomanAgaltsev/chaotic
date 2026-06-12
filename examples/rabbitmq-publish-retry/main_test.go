package main

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	tcrabbitmq "github.com/testcontainers/testcontainers-go/modules/rabbitmq"

	chaosrabbitmq "github.com/RomanAgaltsev/chaotic/adapter/rabbitmq"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestPublishRetrySurvivesOutage(t *testing.T) {
	ctx := context.Background()

	ctr, err := tcrabbitmq.Run(ctx, "rabbitmq:4-management-alpine")
	if err != nil {
		t.Skipf("cannot start RabbitMQ container (Docker unavailable?): %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	amqpURL, err := ctr.AmqpURL(ctx)
	if err != nil {
		t.Fatalf("AmqpURL: %v", err)
	}
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	eng := engine.New()
	cc := chaosrabbitmq.WrapConnection(conn, eng)
	ch, err := cc.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	// durable=true is required on RabbitMQ 4.x — transient (non-durable)
	// non-exclusive queues ("transient_nonexcl_queues") are deprecated and
	// rejected by default.
	q, err := ch.QueueDeclare("retry-demo", true, true, false, false, nil)
	if err != nil {
		t.Fatalf("QueueDeclare: %v", err)
	}

	// Drop the first two publishes: a transient outage. ConnDrop returns
	// amqp.ErrClosed but never touches the real channel, so the retry's next
	// publish lands on the still-open channel.
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.MatchName(q.Name),
		engine.Times(2),
		engine.WithFault(fault.ConnDrop()),
	).Named("outage"))

	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := PublishWithRetry(pctx, ch, "", q.Name, amqp.Publishing{Body: []byte("ok")}, 5); err != nil {
		t.Fatalf("PublishWithRetry failed despite retries: %v", err)
	}
	if got := eng.Hits("outage"); got != 2 {
		t.Fatalf("outage fired %d times, want 2", got)
	}
	msg, ok, err := ch.Get(q.Name, true)
	if err != nil || !ok || string(msg.Body) != "ok" {
		t.Fatalf("Get = %q ok=%v err=%v; want \"ok\"", msg.Body, ok, err)
	}
}
