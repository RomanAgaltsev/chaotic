package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// fakeChannel is a zero-network amqpChannel for unit tests: it records call
// counts and returns preconfigured errors. A zero fakeChannel succeeds on every
// method.
type fakeChannel struct {
	publishErr error
	consumeErr error
	ackErr     error
	nackErr    error

	publishes int
	consumes  int
	acks      int
	nacks     int
}

func (f *fakeChannel) PublishWithContext(_ context.Context, _, _ string, _, _ bool, _ amqp.Publishing) error {
	f.publishes++
	return f.publishErr
}

func (f *fakeChannel) Consume(_, _ string, _, _, _, _ bool, _ amqp.Table) (<-chan amqp.Delivery, error) {
	f.consumes++
	if f.consumeErr != nil {
		return nil, f.consumeErr
	}
	ch := make(chan amqp.Delivery)
	close(ch)
	return ch, nil
}

func (f *fakeChannel) Ack(_ uint64, _ bool) error {
	f.acks++
	return f.ackErr
}

func (f *fakeChannel) Nack(_ uint64, _, _ bool) error {
	f.nacks++
	return f.nackErr
}
