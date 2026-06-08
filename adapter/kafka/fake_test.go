package kafka

import (
	"context"

	kafkago "github.com/segmentio/kafka-go"
)

// fakeReader is a zero-network kafkaReader for unit tests.
type fakeReader struct {
	readErr   error
	fetchErr  error
	commitErr error

	reads   int
	fetches int
	commits int
}

func (f *fakeReader) ReadMessage(context.Context) (kafkago.Message, error) {
	f.reads++
	return kafkago.Message{}, f.readErr
}

func (f *fakeReader) FetchMessage(context.Context) (kafkago.Message, error) {
	f.fetches++
	return kafkago.Message{}, f.fetchErr
}

func (f *fakeReader) CommitMessages(context.Context, ...kafkago.Message) error {
	f.commits++
	return f.commitErr
}

// fakeWriter is a zero-network kafkaWriter for unit tests.
type fakeWriter struct {
	writeErr error
	writes   int
}

func (f *fakeWriter) WriteMessages(context.Context, ...kafkago.Message) error {
	f.writes++
	return f.writeErr
}
