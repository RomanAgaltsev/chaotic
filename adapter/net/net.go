//go:build !chaos_off

package net

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// Conn wraps a net.Conn so the chaotic engine is consulted on each Read and
// Write. Every other net.Conn method is promoted from the embedded conn.
type Conn struct {
	net.Conn
	eng     *engine.Engine
	name    string
	network string
}

// WrapConn returns a net.Conn that consults eng on each Read and Write.
func WrapConn(c net.Conn, eng *engine.Engine) net.Conn {
	name, network := addrInfo(c.RemoteAddr())
	return &Conn{Conn: c, eng: eng, name: name, network: network}
}

func addrInfo(a net.Addr) (name, network string) {
	if a == nil {
		return "", ""
	}
	return a.String(), a.Network()
}

func (c *Conn) op(method string) engine.Op {
	return engine.Op{
		Kind:   engine.OpNet,
		Name:   c.name,
		Method: method,
		Attrs:  map[string]string{"network": c.network},
	}
}

// Read faults the read boundary, then delegates. Read has no context, so a
// latency fault sleeps against a background context.
func (c *Conn) Read(b []byte) (int, error) {
	if !c.eng.Enabled() {
		return c.Conn.Read(b)
	}
	ctx := context.Background()
	action := c.eng.Eval(ctx, c.op("read"))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return 0, mapErr("read", c.network, err)
	}
	n, err := c.Conn.Read(b)
	reportOutcome(ctx, action, err)
	return n, err
}

// Write faults the write boundary, then delegates.
func (c *Conn) Write(b []byte) (int, error) {
	if !c.eng.Enabled() {
		return c.Conn.Write(b)
	}
	ctx := context.Background()
	action := c.eng.Eval(ctx, c.op("write"))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return 0, mapErr("write", c.network, err)
	}
	n, err := c.Conn.Write(b)
	reportOutcome(ctx, action, err)
	return n, err
}

// mapErr translates a fault error into net's model. ConnDrop becomes a
// *net.OpError wrapping io.ErrUnexpectedEOF (a peer reset); Disconnect becomes a
// *net.OpError wrapping io.EOF (an orderly close). Every other fault error
// passes through unchanged.
func mapErr(op, network string, err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return &net.OpError{Op: op, Net: network, Err: io.ErrUnexpectedEOF}
	}
	if errors.Is(err, fault.ErrDisconnect) {
		return &net.OpError{Op: op, Net: network, Err: io.EOF}
	}
	return err
}

// reportOutcome forwards the call's error (or the injected fault) to the engine
// when the action reports outcomes, then runs After to release any held bound
// (e.g. a WithMaxConcurrent slot). Call it exactly once per action. Nil is a no-op.
func reportOutcome(ctx context.Context, action engine.Action, callErr error) {
	if action == nil {
		return
	}
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
	_ = action.After(ctx)
}

// Listener wraps a net.Listener so every accepted connection is chaos-wrapped.
// Other net.Listener methods (Close, Addr) are promoted. Accept itself is not
// faulted in v1.
type Listener struct {
	net.Listener
	eng *engine.Engine
}

// WrapListener returns a net.Listener whose Accept returns chaos-wrapped conns.
func WrapListener(l net.Listener, eng *engine.Engine) net.Listener {
	return &Listener{Listener: l, eng: eng}
}

// Accept accepts a connection and returns it already wrapped.
func (l *Listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return WrapConn(c, l.eng), nil
}

// DialFunc is the dial signature most libraries accept; net.Dialer.DialContext
// satisfies it.
type DialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// Dialer faults at connect time and returns the dialed conn already wrapped, so
// one Dialer covers a whole connection's lifecycle.
type Dialer struct {
	eng  *engine.Engine
	dial DialFunc
}

// WrapDialer returns a *Dialer. inner defaults to (&net.Dialer{}).DialContext
// when nil.
func WrapDialer(eng *engine.Engine, inner DialFunc) *Dialer {
	if inner == nil {
		inner = (&net.Dialer{}).DialContext
	}
	return &Dialer{eng: eng, dial: inner}
}

// DialContext faults the dial (Method "dial", Name the address), then delegates;
// a successful dial returns a chaos-wrapped conn.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if !d.eng.Enabled() {
		c, err := d.dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		return WrapConn(c, d.eng), nil
	}
	op := engine.Op{Kind: engine.OpNet, Name: address, Method: "dial", Attrs: map[string]string{"network": network}}
	action := d.eng.Eval(ctx, op)
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return nil, mapErr("dial", network, err)
	}
	c, err := d.dial(ctx, network, address)
	reportOutcome(ctx, action, err)
	if err != nil {
		return nil, err
	}
	return WrapConn(c, d.eng), nil
}
