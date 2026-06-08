package kafka

import (
	"context"

	kafkago "github.com/segmentio/kafka-go"
)

// kafkaReader is the faultable subset of *kafka.Reader. *kafka.Reader satisfies
// it directly. Unit tests supply a fake. Every other *kafka.Reader method (Stats,
// Lag, Offset, Close, SetOffset, ...) reaches callers via embedded value.
type kafkaReader interface {
	ReadMessage(ctx context.Context) (kafkago.Message, error)
	FetchMessage(ctx context.Context) (kafkago.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafkago.Message) error
}

// kafkaWriter is the faultable subset of *kafka.Writer. *kafka.Writer satisfies it
// directly; unit tests supply a fake.
type kafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafkago.Message) error
}
