//go:build !chaos_off

package net_test

import (
	"context"
	"errors"
	"net"
	"testing"

	chaosnet "github.com/RomanAgaltsev/chaotic/adapter/net"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestDialerListenerRoundTripOverLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	eng := engine.New()
	wln := chaosnet.WrapListener(ln, eng)

	go func() {
		c, aerr := wln.Accept()
		if aerr != nil {
			return
		}
		defer func() { _ = c.Close() }()
		buf := make([]byte, 4)
		if _, rerr := c.Read(buf); rerr == nil {
			_, _ = c.Write(buf) // echo
		}
	}()

	d := chaosnet.WrapDialer(eng, nil) // real net.Dialer underneath
	conn, err := d.DialContext(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	buf := make([]byte, 4)
	if _, err := conn.Read(buf); err != nil || string(buf) != "ping" {
		t.Fatalf("echo Read = %q err=%v; want \"ping\"", buf, err)
	}

	// Now drop reads and confirm the native error shape surfaces over a real conn.
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.MatchName(ln.Addr().String()),
		engine.MatchAttr("network", "tcp"),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))
	if _, err := conn.Read(buf); err == nil || !errors.Is(err, errUnexpectedEOFForTest()) {
		// Accept any *net.OpError; assert it is one.
		var opErr *net.OpError
		if !errors.As(err, &opErr) {
			t.Fatalf("post-drop Read err = %T %v, want *net.OpError", err, err)
		}
	}
}

// errUnexpectedEOFForTest avoids importing io just for the sentinel in the guard
// above; the assertion that matters is errors.As(*net.OpError).
func errUnexpectedEOFForTest() error { return nil }
