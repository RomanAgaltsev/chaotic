//go:build chaos_off

// Package net (chaos_off build): the wrappers add no behavior and no allocation.
// WrapConn returns the conn unchanged; WrapListener returns the listener; the
// Dialer just dials.
package net

import (
	"context"
	"net"

	"github.com/RomanAgaltsev/chaotic/engine"
)

func WrapConn(c net.Conn, _ *engine.Engine) net.Conn { return c }

func WrapListener(l net.Listener, _ *engine.Engine) net.Listener { return l }

type DialFunc func(ctx context.Context, network, address string) (net.Conn, error)

type Dialer struct {
	dial DialFunc
}

func WrapDialer(_ *engine.Engine, inner DialFunc) *Dialer {
	if inner == nil {
		inner = (&net.Dialer{}).DialContext
	}
	return &Dialer{dial: inner}
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.dial(ctx, network, address)
}
