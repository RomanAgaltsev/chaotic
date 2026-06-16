//go:build !chaos_off

package rabbitmq

import (
	"context"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// Channel wraps an *amqp.Channel so the chaotic engine is consulted on each
// publish, consume and ack/nack. Every other *amqp.Channel method is promoted
// from the embedded value, so a *Channel is a drop-in for *amqp.Channel.
//
// In production the embedded *amqp.Channel and the ch seam point at the same
// channel: the embedded value supplies method promotion for the un-faulted
// methods, while the faultable overrides call through ch. The seam exists so
// unit tests can drive the overrides against a fake without a live broker; the
// two must always reference the same underlying channel.
type Channel struct {
	*amqp.Channel             // promotes every un-faulted method (QueueDeclare, Get, ...)
	ch            amqpChannel // faultable-method seam; == Channel in production, a fake in tests
	eng           *engine.Engine
}

// WrapChannel returns a *Channel that consults eng on each faultable operation.
func WrapChannel(ch *amqp.Channel, eng *engine.Engine) *Channel {
	return &Channel{
		Channel: ch,
		ch:      ch,
		eng:     eng,
	}
}

// PublishWithContext faults the publish path, then delegates to the real channel.
func (c *Channel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	if !c.eng.Enabled() {
		return c.ch.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
	}
	op := engine.Op{
		Kind:   engine.OpRabbitMQ,
		Name:   key,
		Method: "publish",
		Attrs: map[string]string{
			"exchange":    exchange,
			"routing_key": key,
		},
	}
	action := c.eng.Eval(ctx, op)
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return mapErr(err)
	}
	err := c.ch.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
	reportOutcome(ctx, action, err)
	return err
}

// Publish is the deprecated context-less publish; it routes through
// PublishWithContext with a background context so chaos fires identically.
func (c *Channel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return c.PublishWithContext(context.Background(), exchange, key, mandatory, immediate, msg)
}

// Consume faults once at subscription open, then delegates. Per-delivery faults
// are deferred until the per-row primitive lands.
// Consume has no context parameter, so latency faults sleep against a background
// context.
func (c *Channel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if !c.eng.Enabled() {
		return c.ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	}
	ctx := context.Background()
	op := engine.Op{
		Kind:   engine.OpRabbitMQ,
		Name:   queue,
		Method: "consume",
		Attrs:  map[string]string{"queue": queue},
	}
	action := c.eng.Eval(ctx, op)
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return nil, mapErr(err)
	}
	deliveries, err := c.ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	reportOutcome(ctx, action, err)
	return deliveries, err
}

// Ack faults the channel-level acknowledgement. NOTE: a Delivery returned by
// Consume carries its own Acknowledger bound to the underlying *amqp.Channel, so
// delivery.Ack() bypasses chaos — per-delivery acknowledgement faulting is
// deferred with the per-row primitive. Call this channel-level Ack to exercise
// the path.
func (c *Channel) Ack(tag uint64, multiple bool) error {
	if !c.eng.Enabled() {
		return c.ch.Ack(tag, multiple)
	}
	ctx := context.Background()
	op := engine.Op{Kind: engine.OpRabbitMQ, Name: "ack", Method: "ack"}
	action := c.eng.Eval(ctx, op)
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return mapErr(err)
	}
	err := c.ch.Ack(tag, multiple)
	reportOutcome(ctx, action, err)
	return err
}

// Nack faults the channel-level negative acknowledgement. The delivery-level
// caveat on Ack applies here too.
func (c *Channel) Nack(tag uint64, multiple, requeue bool) error {
	if !c.eng.Enabled() {
		return c.ch.Nack(tag, multiple, requeue)
	}
	ctx := context.Background()
	op := engine.Op{Kind: engine.OpRabbitMQ, Name: "nack", Method: "nack"}
	action := c.eng.Eval(ctx, op)
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return mapErr(err)
	}
	err := c.ch.Nack(tag, multiple, requeue)
	reportOutcome(ctx, action, err)
	return err
}

// mapErr translates a fault error into amqp091-go's native error model. ConnDrop
// becomes amqp.ErrClosed (an *amqp.Error) so the caller's channel/connection
// recovery path engages; every other fault error passes through unchanged, so a
// caller who wants a specific broker error supplies fault.Error(&amqp.Error{...}).
func mapErr(err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return amqp.ErrClosed
	}
	return err
}

// reportOutcome forwards the call's error (or the injected fault) to the engine
// when the action reports outcomes, then runs After to release any held bound
// (e.g. a WithMaxConcurrent slot). Call it exactly once per action, or the slot
// leaks and the failure budget never sees the call. A nil action is a no-op.
func reportOutcome(ctx context.Context, action engine.Action, callErr error) {
	if action == nil {
		return
	}
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
	_ = action.After(ctx)
}

// Conn wraps an *amqp.Connection so chaos can fire when a new channel is opened
// and so channels it returns are already wrapped. Every other *amqp.Connection
// method is promoted from the embedded value.
type Conn struct {
	*amqp.Connection
	eng *engine.Engine
}

// WrapConnection returns a *Conn whose Channel opens chaos-wrapped channels.
func WrapConnection(conn *amqp.Connection, eng *engine.Engine) *Conn {
	return &Conn{
		Connection: conn,
		eng:        eng,
	}
}

// Channel opens a new AMQP channel and returns it already wrapped, so publishes
// and consumes on it are faulted without a second WrapChannel call. Chaos also
// fires at open (Method "channel"), modeling a connection that can no longer open
// channels.
func (c *Conn) Channel() (*Channel, error) {
	if !c.eng.Enabled() {
		ch, err := c.Connection.Channel()
		if err != nil {
			return nil, err
		}
		return WrapChannel(ch, c.eng), nil
	}
	ctx := context.Background()
	op := engine.Op{Kind: engine.OpRabbitMQ, Name: "channel", Method: "channel"}
	action := c.eng.Eval(ctx, op)
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return nil, mapErr(err)
	}
	ch, err := c.Connection.Channel()
	reportOutcome(ctx, action, err)
	if err != nil {
		return nil, err
	}
	return WrapChannel(ch, c.eng), nil
}
