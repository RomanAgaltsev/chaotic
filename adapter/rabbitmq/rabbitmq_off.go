//go:build chaos_off

// Package rabbitmq (chaos_off build): the wrapper adds no behavior and no
// allocation. Faultable methods forward straight to the wrapped channel; every
// other method is promoted from the embedded *amqp.Channel / *amqp.Connection.
package rabbitmq

import (
	"context"

	"github.com/RomanAgaltsev/chaotic/engine"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Channel struct {
	*amqp.Channel
	ch amqpChannel
}

func WrapChannel(ch *amqp.Channel, _ *engine.Engine) *Channel {
	return &Channel{Channel: ch, ch: ch}
}

func (c *Channel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return c.ch.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
}

func (c *Channel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return c.ch.PublishWithContext(context.Background(), exchange, key, mandatory, immediate, msg)
}

func (c *Channel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return c.ch.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

func (c *Channel) Ack(tag uint64, multiple bool) error { return c.ch.Ack(tag, multiple) }
func (c *Channel) Nack(tag uint64, multiple, requeue bool) error {
	return c.ch.Nack(tag, multiple, requeue)
}

type Conn struct {
	*amqp.Connection
}

func WrapConnection(conn *amqp.Connection, _ *engine.Engine) *Conn {
	return &Conn{Connection: conn}
}

func (c *Conn) Channel() (*Channel, error) {
	ch, err := c.Connection.Channel()
	if err != nil {
		return nil, err
	}
	return WrapChannel(ch, nil), nil
}
