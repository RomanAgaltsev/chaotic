//go:build chaos_off

// Package nats (chaos_off build): the wrapper adds no behavior and no allocation.
// Faulted methods forward straight to the wrapped connection; Option returns a
// no-op nats.Option; every other method is promoted from the embedded *nats.Conn.
package nats

import (
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/RomanAgaltsev/chaotic/engine"
)

type Conn struct {
	*natsgo.Conn
	conn natsConn
}

func WrapConn(nc *natsgo.Conn, _ *engine.Engine) *Conn {
	return &Conn{Conn: nc, conn: nc}
}

func (c *Conn) Publish(subj string, data []byte) error { return c.conn.Publish(subj, data) }

func (c *Conn) Request(subj string, data []byte, timeout time.Duration) (*natsgo.Msg, error) {
	return c.conn.Request(subj, data, timeout)
}

func (c *Conn) Subscribe(subj string, cb natsgo.MsgHandler) (*natsgo.Subscription, error) {
	return c.conn.Subscribe(subj, cb)
}

func (c *Conn) QueueSubscribe(subj, queue string, cb natsgo.MsgHandler) (*natsgo.Subscription, error) {
	return c.conn.QueueSubscribe(subj, queue, cb)
}

func (c *Conn) Drain() error { return c.conn.Drain() }

// Option is a no-op nats.Option under chaos_off: it installs no custom dialer.
func Option(_ *engine.Engine) natsgo.Option {
	return func(*natsgo.Options) error { return nil }
}
