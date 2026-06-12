//go:build integration && !chaos_off

package rabbitmq_test

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	tcrabbitmq "github.com/testcontainers/testcontainers-go/modules/rabbitmq"

	chaosrabbitmq "github.com/RomanAgaltsev/chaotic/adapter/rabbitmq"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestWrapperAgainstRealBroker(t *testing.T) {
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
	ch, err := cc.Channel() // auto-wrapped *Channel
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}

	// Promoted (un-faulted) method: declare a queue.
	// NOTE: durable=true is required on RabbitMQ 4.x — transient (non-durable)
	// non-exclusive queues ("transient_nonexcl_queues") are deprecated and
	// rejected by default.
	q, err := ch.QueueDeclare("chaos-test", true, true, false, false, nil)
	if err != nil {
		t.Fatalf("QueueDeclare: %v", err)
	}

	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Happy path: publish, then Get it back (Get is promoted, un-faulted).
	if err := ch.PublishWithContext(pctx, "", q.Name, false, false, amqp.Publishing{Body: []byte("hello")}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	msg, ok, err := ch.Get(q.Name, true)
	if err != nil || !ok || string(msg.Body) != "hello" {
		t.Fatalf("Get = %q ok=%v err=%v; want \"hello\"", msg.Body, ok, err)
	}

	// Fault path: a rule that fails the publish surfaces as the supplied *amqp.Error.
	sentinel := &amqp.Error{Code: amqp.ConnectionForced, Reason: "chaos"}
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.MatchName(q.Name),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("pub-fail"))

	err = ch.PublishWithContext(pctx, "", q.Name, false, false, amqp.Publishing{Body: []byte("hello")})
	var aerr *amqp.Error
	if !errors.As(err, &aerr) {
		t.Fatalf("publish err = %T %v, want *amqp.Error", err, err)
	}
}
