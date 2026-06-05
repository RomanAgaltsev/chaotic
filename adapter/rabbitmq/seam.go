package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// amqpChannel is the faultable subset of *amqp.Channel that the chaos wrapper
// intercepts. *amqp.Channel satisfies it directly. Unit tests supply a fake.
// Every other *amqp.Channel method (QueueDeclare, ExchangeDeclare, Qos, Get, ...)
// reaches callers unchanged via embedded *amqpChannel in Channel.
type amqpChannel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Ack(tag uint64, multiple bool) error
	Nack(tag uint64, multiple, requeue bool) error
}
