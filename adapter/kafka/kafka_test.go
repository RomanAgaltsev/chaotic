//go:build !chaos_off

package kafka

import (
	"context"
	"errors"
	"io"
	"testing"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func newWriter(fake kafkaWriter, eng *engine.Engine) *Writer {
	return &Writer{w: fake, eng: eng, topic: "events"}
}

func TestWriteMessagesInjectsError(t *testing.T) {
	sentinel := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpKafka),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("err"))

	fake := &fakeWriter{}
	w := newWriter(fake, eng)
	err := w.WriteMessages(context.Background(), kafkago.Message{Value: []byte("a")}, kafkago.Message{Value: []byte("b")})

	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
	if fake.writes != 0 {
		t.Fatalf("underlying WriteMessages ran %d times, want 0 (fault short-circuits)", fake.writes)
	}
}

func TestWriteMessagesFaultsOncePerCall(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpKafka),
		engine.Always(),
		engine.WithFault(fault.Error(errors.New("x"))),
	).Named("once"))

	w := newWriter(&fakeWriter{}, eng)
	// A batch of three messages must fire the rule exactly once, not per message.
	_ = w.WriteMessages(context.Background(),
		kafkago.Message{Value: []byte("a")},
		kafkago.Message{Value: []byte("b")},
		kafkago.Message{Value: []byte("c")},
	)
	if got := eng.Hits("once"); got != 1 {
		t.Fatalf("rule fired %d times for a 3-message batch, want 1 (once per call)", got)
	}
}

func TestWriteMessagesConnDropMapsToUnexpectedEOF(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpKafka),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	w := newWriter(&fakeWriter{}, eng)
	err := w.WriteMessages(context.Background(), kafkago.Message{Value: []byte("a")})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestWriteMessagesPassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeWriter{}
	w := newWriter(fake, engine.New())
	if err := w.WriteMessages(context.Background(), kafkago.Message{Value: []byte("a")}); err != nil {
		t.Fatalf("err = %v, want nil passthrough", err)
	}
	if fake.writes != 1 {
		t.Fatalf("underlying WriteMessages ran %d times, want 1", fake.writes)
	}
}

func newReader(fake kafkaReader, eng *engine.Engine) *Reader {
	return &Reader{r: fake, eng: eng, topic: "events", attrs: map[string]string{"group": "g"}}
}

func TestReadFetchCommitConnDropMapsToUnexpectedEOF(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpKafka),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	fake := &fakeReader{}
	r := newReader(fake, eng)
	ctx := context.Background()

	if _, err := r.ReadMessage(ctx); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("ReadMessage err = %v, want io.ErrUnexpectedEOF", err)
	}
	if _, err := r.FetchMessage(ctx); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("FetchMessage err = %v, want io.ErrUnexpectedEOF", err)
	}
	if err := r.CommitMessages(ctx, kafkago.Message{}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("CommitMessages err = %v, want io.ErrUnexpectedEOF", err)
	}
	if fake.reads != 0 || fake.fetches != 0 || fake.commits != 0 {
		t.Fatalf("an underlying op ran despite the fault: reads=%d fetches=%d commits=%d", fake.reads, fake.fetches, fake.commits)
	}
}

func TestReadFetchCommitFault(t *testing.T) {
	sentinel := errors.New("consumer down")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpKafka),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("consume-err"))

	fake := &fakeReader{}
	r := newReader(fake, eng)
	ctx := context.Background()

	if _, err := r.ReadMessage(ctx); !errors.Is(err, sentinel) {
		t.Fatalf("ReadMessage err = %v, want sentinel", err)
	}
	if _, err := r.FetchMessage(ctx); !errors.Is(err, sentinel) {
		t.Fatalf("FetchMessage err = %v, want sentinel", err)
	}
	if err := r.CommitMessages(ctx, kafkago.Message{}); !errors.Is(err, sentinel) {
		t.Fatalf("CommitMessages err = %v, want sentinel", err)
	}
	if fake.reads != 0 || fake.fetches != 0 || fake.commits != 0 {
		t.Fatalf("an underlying op ran despite the fault: reads=%d fetches=%d commits=%d", fake.reads, fake.fetches, fake.commits)
	}
}

func TestReaderPassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeReader{}
	r := newReader(fake, engine.New())
	ctx := context.Background()
	_, _ = r.ReadMessage(ctx)
	_, _ = r.FetchMessage(ctx)
	_ = r.CommitMessages(ctx, kafkago.Message{})
	if fake.reads != 1 || fake.fetches != 1 || fake.commits != 1 {
		t.Fatalf("passthrough miscount: reads=%d fetches=%d commits=%d, want all 1", fake.reads, fake.fetches, fake.commits)
	}
}
