//go:build !chaos_off

package nats

import (
	"context"
	"errors"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func newConn(fake natsConn, eng *engine.Engine) *Conn {
	return &Conn{conn: fake, eng: eng}
}

func TestPublishInjectsError(t *testing.T) {
	sentinel := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNATS),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("err"))

	fake := &fakeConn{}
	c := newConn(fake, eng)
	err := c.Publish("events", []byte("hi"))

	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
	if fake.publishes != 0 {
		t.Fatalf("underlying Publish ran %d times, want 0 (fault short-circuits)", fake.publishes)
	}
}

func TestPublishConnDropMapsToConnectionClosed(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNATS),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	c := newConn(&fakeConn{}, eng)
	err := c.Publish("events", []byte("hi"))
	if !errors.Is(err, natsgo.ErrConnectionClosed) {
		t.Fatalf("err = %v, want nats.ErrConnectionClosed", err)
	}
}

func TestPublishPassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeConn{}
	c := newConn(fake, engine.New()) // no rules
	if err := c.Publish("events", []byte("hi")); err != nil {
		t.Fatalf("err = %v, want nil passthrough", err)
	}
	if fake.publishes != 1 {
		t.Fatalf("underlying Publish ran %d times, want 1", fake.publishes)
	}
	_ = context.Background()
}

func TestRequestAndDrainFault(t *testing.T) {
	sentinel := errors.New("no responder")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNATS),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("req-err")) // matches every OpNATS op

	fake := &fakeConn{}
	c := newConn(fake, eng)

	if _, err := c.Request("svc.echo", []byte("ping"), time.Second); !errors.Is(err, sentinel) {
		t.Fatalf("Request err = %v, want sentinel", err)
	}
	if err := c.Drain(); !errors.Is(err, sentinel) {
		t.Fatalf("Drain err = %v, want sentinel", err)
	}
	if fake.requests != 0 || fake.drains != 0 {
		t.Fatalf("an underlying op ran despite the fault: requests=%d drains=%d", fake.requests, fake.drains)
	}
}

func TestRequestPassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeConn{}
	c := newConn(fake, engine.New())
	if _, err := c.Request("svc.echo", []byte("ping"), time.Second); err != nil {
		t.Fatalf("Request err = %v, want nil passthrough", err)
	}
	if fake.requests != 1 {
		t.Fatalf("underlying Request ran %d times, want 1", fake.requests)
	}
}

func TestSubscribeFaultsAtOpen(t *testing.T) {
	sentinel := errors.New("subscribe refused")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNATS),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("sub-err"))

	fake := &fakeConn{}
	c := newConn(fake, eng)

	if _, err := c.Subscribe("events", func(*natsgo.Msg) {}); !errors.Is(err, sentinel) {
		t.Fatalf("Subscribe err = %v, want sentinel", err)
	}
	if _, err := c.QueueSubscribe("events", "workers", func(*natsgo.Msg) {}); !errors.Is(err, sentinel) {
		t.Fatalf("QueueSubscribe err = %v, want sentinel", err)
	}
	if fake.subscribes != 0 || fake.queues != 0 {
		t.Fatalf("an underlying subscribe ran despite the fault: subscribes=%d queues=%d", fake.subscribes, fake.queues)
	}
}

func TestSubscribePassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeConn{}
	c := newConn(fake, engine.New())
	_, _ = c.Subscribe("events", func(*natsgo.Msg) {})
	_, _ = c.QueueSubscribe("events", "workers", func(*natsgo.Msg) {})
	if fake.subscribes != 1 || fake.queues != 1 {
		t.Fatalf("passthrough miscount: subscribes=%d queues=%d, want both 1", fake.subscribes, fake.queues)
	}
}
