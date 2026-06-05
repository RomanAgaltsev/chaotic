//go:build !chaos_off

package rabbitmq

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func newChannel(fake amqpChannel, eng *engine.Engine) *Channel {
	return &Channel{
		ch:  fake,
		eng: eng,
	}
}

func TestPublishInjectsError(t *testing.T) {
	sentinel := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("err"))

	fake := &fakeChannel{}
	c := newChannel(fake, eng)
	err := c.PublishWithContext(context.Background(), "ex", "rk", false, false, amqp.Publishing{})

	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
	if fake.publishes != 0 {
		t.Fatalf("underlying publish ran %d times, want 0 (fault short-circuits)", fake.publishes)
	}
}

func TestPublishConnDropMapsToErrClosed(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	c := newChannel(&fakeChannel{}, eng)
	err := c.PublishWithContext(context.Background(), "ex", "rk", false, false, amqp.Publishing{})

	if !errors.Is(err, amqp.ErrClosed) {
		t.Fatalf("err = %v, want amqp.ErrClosed", err)
	}
}

func TestPublishPassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeChannel{}
	c := newChannel(fake, engine.New()) // no rules
	if err := c.PublishWithContext(context.Background(), "ex", "rk", false, false, amqp.Publishing{}); err != nil {
		t.Fatalf("err = %v, want nil passthrough", err)
	}
	if fake.publishes != 1 {
		t.Fatalf("underlying publish ran %d times, want 1", fake.publishes)
	}
}

func TestDeprecatedPublishRoutesThroughChaos(t *testing.T) {
	sentinel := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("err"))

	c := newChannel(&fakeChannel{}, eng)
	// exercising the context-less Publish on purpose
	err := c.Publish("ex", "rk", false, false, amqp.Publishing{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Publish err = %v, want sentinel (deprecated form must fault too)", err)
	}
}

func TestConsumeFaultsAtOpen(t *testing.T) {
	sentinel := errors.New("no consumer")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.MatchName("jobs"),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("consume-err"))

	fake := &fakeChannel{}
	c := newChannel(fake, eng)
	deliveries, err := c.Consume("jobs", "worker-1", false, false, false, false, nil)

	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
	if deliveries != nil {
		t.Fatalf("deliveries = %v, want nil when consume faults at open", deliveries)
	}
	if fake.consumes != 0 {
		t.Fatalf("underlying Consume ran %d times, want 0 (fault short-circuits)", fake.consumes)
	}
}

func TestConsumePassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeChannel{}
	c := newChannel(fake, engine.New())
	deliveries, err := c.Consume("jobs", "worker-1", false, false, false, false, nil)
	if err != nil {
		t.Fatalf("err = %v, want nil passthrough", err)
	}
	if deliveries == nil {
		t.Fatal("deliveries = nil, want the channel from the underlying Consume")
	}
	if fake.consumes != 1 {
		t.Fatalf("underlying Consume ran %d times, want 1", fake.consumes)
	}
}

func TestAckAndNackFault(t *testing.T) {
	sentinel := errors.New("ack failed")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("ack-err"))

	fake := &fakeChannel{}
	c := newChannel(fake, eng)

	if err := c.Ack(1, false); !errors.Is(err, sentinel) {
		t.Fatalf("Ack err = %v, want sentinel", err)
	}
	if err := c.Nack(2, false, true); !errors.Is(err, sentinel) {
		t.Fatalf("Nack err = %v, want sentinel", err)
	}
	if fake.acks != 0 || fake.nacks != 0 {
		t.Fatalf("underlying acks=%d nacks=%d, want 0/0 (faults short-circuit)", fake.acks, fake.nacks)
	}
}

func TestAckPassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeChannel{}
	c := newChannel(fake, engine.New())
	if err := c.Ack(1, false); err != nil {
		t.Fatalf("Ack err = %v, want nil passthrough", err)
	}
	if fake.acks != 1 {
		t.Fatalf("underlying Ack ran %d times, want 1", fake.acks)
	}
}
