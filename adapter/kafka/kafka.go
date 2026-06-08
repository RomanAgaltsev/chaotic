//go:build !chaos_off

package kafka

import (
	"context"
	"errors"
	"io"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// Writer wraps a *kafka.Writer so the chaotic engine is consulted on each
// WriteMessages call. Every other *kafka.Writer method is promoted from the
// embedded value, so a *Writer is a drop-in for *kafka.Writer.
type Writer struct {
	*kafkago.Writer
	w     kafkaWriter
	eng   *engine.Engine
	topic string
}

// WrapWriter returns a *Writer that consults eng on each WriteMessages call. The
// Op's Name is the writer's topic (empty if the topic is set per-message).
func WrapWriter(w *kafkago.Writer, eng *engine.Engine) *Writer {
	return &Writer{Writer: w, w: w, eng: eng, topic: w.Topic}
}

func (w *Writer) op() engine.Op {
	return engine.Op{
		Kind:   engine.OpKafka,
		Name:   w.topic,
		Method: "write",
	}
}

// WriteMessages faults the publish path once per call (not per message), then
// delegates to the real writer.
func (w *Writer) WriteMessages(ctx context.Context, msgs ...kafkago.Message) error {
	if !w.eng.Enabled() {
		return w.w.WriteMessages(ctx, msgs...)
	}
	action := w.eng.Eval(ctx, w.op())
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return mapErr(err)
	}
	err := w.w.WriteMessages(ctx, msgs...)
	reportOutcome(ctx, action, err)
	return err
}

// Reader wraps a *kafka.Reader so the chaotic engine is consulted on each read,
// fetch, and commit. Every other *kafka.Reader method is promoted from the
// embedded value, so a *Reader is a drop-in for *kafka.Reader.
type Reader struct {
	*kafkago.Reader
	r     kafkaReader
	eng   *engine.Engine
	topic string
	attrs map[string]string
}

// WrapReader returns a *Reader that consults eng on each read/fetch/commit. The
// Op's Name is the reader's topic and Attrs carry the consumer group. Topic and
// group are immutable from here on, so the Attrs map is built once and shared
// across calls (the engine only reads it).
func WrapReader(r *kafkago.Reader, eng *engine.Engine) *Reader {
	cfg := r.Config()
	return &Reader{Reader: r, r: r, eng: eng, topic: cfg.Topic, attrs: map[string]string{"group": cfg.GroupID}}
}

func (r *Reader) op(method string) engine.Op {
	return engine.Op{
		Kind:   engine.OpKafka,
		Name:   r.topic,
		Method: method,
		Attrs:  r.attrs,
	}
}

// ReadMessage faults the blocking read (auto-commit consumer), then delegates.
func (r *Reader) ReadMessage(ctx context.Context) (kafkago.Message, error) {
	if !r.eng.Enabled() {
		return r.r.ReadMessage(ctx)
	}
	action := r.eng.Eval(ctx, r.op("read"))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return kafkago.Message{}, mapErr(err)
	}
	msg, err := r.r.ReadMessage(ctx)
	reportOutcome(ctx, action, err)
	return msg, err
}

// FetchMessage faults the manual-commit read, then delegates.
func (r *Reader) FetchMessage(ctx context.Context) (kafkago.Message, error) {
	if !r.eng.Enabled() {
		return r.r.FetchMessage(ctx)
	}
	action := r.eng.Eval(ctx, r.op("fetch"))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return kafkago.Message{}, mapErr(err)
	}
	msg, err := r.r.FetchMessage(ctx)
	reportOutcome(ctx, action, err)
	return msg, err
}

// CommitMessages faults the offset commit, then delegates.
func (r *Reader) CommitMessages(ctx context.Context, msgs ...kafkago.Message) error {
	if !r.eng.Enabled() {
		return r.r.CommitMessages(ctx, msgs...)
	}
	action := r.eng.Eval(ctx, r.op("commit"))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return mapErr(err)
	}
	err := r.r.CommitMessages(ctx, msgs...)
	reportOutcome(ctx, action, err)
	return err
}

// mapErr translates a fault error into kafka-go's native model. ConnDrop becomes
// io.ErrUnexpectedEOF, which kafka-go classifies as a transport error and retries;
// every other fault error passes through unchanged.
func mapErr(err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return io.ErrUnexpectedEOF
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
