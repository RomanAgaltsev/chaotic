//go:build !chaos_off

package nats

import (
	"context"
	"errors"
	"net"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// Conn wraps a *nats.Conn so the chaotic engine is consulted on each publish,
// request, subscribe, and drain. Every other *nats.Conn method is promoted from
// the embedded value, so a *Conn is a drop-in for *nats.Conn.
type Conn struct {
	*natsgo.Conn
	conn natsConn
	eng  *engine.Engine
}

// WrapConn returns a *Conn that consults eng on each faultable per-call method.
func WrapConn(nc *natsgo.Conn, eng *engine.Engine) *Conn {
	return &Conn{Conn: nc, conn: nc, eng: eng}
}

// op builds the Op for a subject + method. Subscriptions optionally carry a queue.
func op(subject, method, queue string) engine.Op {
	o := engine.Op{Kind: engine.OpNATS, Name: subject, Method: method}
	if queue != "" {
		o.Attrs = map[string]string{"queue": queue}
	}
	return o
}

// Publish faults the publish path, then delegates. Publish has no context, so a
// latency fault sleeps against a background context.
func (c *Conn) Publish(subj string, data []byte) error {
	if !c.eng.Enabled() {
		return c.conn.Publish(subj, data)
	}
	ctx := context.Background()
	action := c.eng.Eval(ctx, op(subj, "publish", ""))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return mapErr(err)
	}
	err := c.conn.Publish(subj, data)
	reportOutcome(ctx, action, err)
	return err
}

// mapErr translates a fault error into nats.go's native model. ConnDrop becomes
// the transient nats.ErrConnectionClosed so the caller's reconnect/retry path
// engages; the wrapper never calls nc.Close(). Every other fault error passes
// through unchanged.
func mapErr(err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return natsgo.ErrConnectionClosed
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

// Request faults the request/reply path, then delegates. timeout is the caller's
// reply deadline; the chaos latency fault (if any) sleeps before the real request.
func (c *Conn) Request(subj string, data []byte, timeout time.Duration) (*natsgo.Msg, error) {
	if !c.eng.Enabled() {
		return c.conn.Request(subj, data, timeout)
	}
	ctx := context.Background()
	action := c.eng.Eval(ctx, op(subj, "request", ""))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return nil, mapErr(err)
	}
	msg, err := c.conn.Request(subj, data, timeout)
	reportOutcome(ctx, action, err)
	return msg, err
}

// Drain faults the connection drain, then delegates. Note: Drain is the graceful
// shutdown path; a chaos fault here models drain failing, not the connection
// closing.
func (c *Conn) Drain() error {
	if !c.eng.Enabled() {
		return c.conn.Drain()
	}
	ctx := context.Background()
	action := c.eng.Eval(ctx, op("", "drain", ""))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return mapErr(err)
	}
	err := c.conn.Drain()
	reportOutcome(ctx, action, err)
	return err
}

// Subscribe faults at subscription open, then delegates. Per-delivery faults are
// deferred until the per-row primitive lands (see doc.go / design §4.4).
func (c *Conn) Subscribe(subj string, cb natsgo.MsgHandler) (*natsgo.Subscription, error) {
	if !c.eng.Enabled() {
		return c.conn.Subscribe(subj, cb)
	}
	ctx := context.Background()
	action := c.eng.Eval(ctx, op(subj, "subscribe", ""))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return nil, mapErr(err)
	}
	sub, err := c.conn.Subscribe(subj, cb)
	reportOutcome(ctx, action, err)
	return sub, err
}

// QueueSubscribe faults at queue-subscription open, then delegates. The queue
// group is carried in the Op's Attrs.
func (c *Conn) QueueSubscribe(subj, queue string, cb natsgo.MsgHandler) (*natsgo.Subscription, error) {
	if !c.eng.Enabled() {
		return c.conn.QueueSubscribe(subj, queue, cb)
	}
	ctx := context.Background()
	action := c.eng.Eval(ctx, op(subj, "subscribe", queue))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return nil, mapErr(err)
	}
	sub, err := c.conn.QueueSubscribe(subj, queue, cb)
	reportOutcome(ctx, action, err)
	return sub, err
}

// Option returns a nats.Option that installs a chaos dialer. The dialer consults
// eng on every Dial — including reconnect attempts — so ConnDrop/Error can model
// a NATS server that is unreachable, exercising nats.go's reconnection logic. Pass
// it to nats.Connect alongside your other options.
func Option(eng *engine.Engine) natsgo.Option {
	return natsgo.SetCustomDialer(&chaosDialer{eng: eng})
}

// chaosDialer is a nats.CustomDialer that faults the connection establishment.
type chaosDialer struct {
	eng *engine.Engine
}

// Dial consults the engine for the target address, then dials for real. A fault
// returns before any socket is opened, so the connection is never half-formed.
func (d *chaosDialer) Dial(network, address string) (net.Conn, error) {
	if !d.eng.Enabled() {
		return net.Dial(network, address)
	}
	ctx := context.Background()
	action := d.eng.Eval(ctx, op(address, "dial", ""))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return nil, mapErr(err)
	}
	conn, err := net.Dial(network, address)
	reportOutcome(ctx, action, err)
	return conn, err
}
