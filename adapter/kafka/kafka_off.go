//go:build chaos_off

// Package kafka (chaos_off build): the wrapper adds no behavior and no
// allocation. Faulted methods forward straight to the wrapped reader/writer; every
// other method is promoted from the embedded *kafka.Reader / *kafka.Writer.
package kafka

import (
	"context"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/RomanAgaltsev/chaotic/engine"
)

type Writer struct {
	*kafkago.Writer
	w kafkaWriter
}

func WrapWriter(w *kafkago.Writer, _ *engine.Engine) *Writer {
	return &Writer{Writer: w, w: w}
}

func (w *Writer) WriteMessages(ctx context.Context, msgs ...kafkago.Message) error {
	return w.w.WriteMessages(ctx, msgs...)
}

type Reader struct {
	*kafkago.Reader
	r kafkaReader
}

func WrapReader(r *kafkago.Reader, _ *engine.Engine) *Reader {
	return &Reader{Reader: r, r: r}
}

func (r *Reader) ReadMessage(ctx context.Context) (kafkago.Message, error) {
	return r.r.ReadMessage(ctx)
}

func (r *Reader) FetchMessage(ctx context.Context) (kafkago.Message, error) {
	return r.r.FetchMessage(ctx)
}

func (r *Reader) CommitMessages(ctx context.Context, msgs ...kafkago.Message) error {
	return r.r.CommitMessages(ctx, msgs...)
}
