//go:build !chaos_off

package net_test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	chaosnet "github.com/RomanAgaltsev/chaotic/adapter/net"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// pipePair returns a connected in-memory conn pair.
func pipePair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	c1, c2 := net.Pipe()
	t.Cleanup(func() { _ = c1.Close(); _ = c2.Close() })
	return c1, c2
}

func TestReadInjectsError(t *testing.T) {
	sentinel := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.MatchAttr("network", "pipe"), // net.Pipe conns report network "pipe"
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("err"))

	a, _ := pipePair(t)
	c := chaosnet.WrapConn(a, eng)
	n, err := c.Read(make([]byte, 8))
	if n != 0 || !errors.Is(err, sentinel) {
		t.Fatalf("Read = (%d, %v), want (0, sentinel)", n, err)
	}
}

func TestReadConnDropMapsToNetOpError(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	a, _ := pipePair(t)
	c := chaosnet.WrapConn(a, eng)
	_, err := c.Read(make([]byte, 8))
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("err = %T %v, want *net.OpError", err, err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err chain = %v, want io.ErrUnexpectedEOF inside", err)
	}
}

func TestWritePassesThroughWhenNoRule(t *testing.T) {
	eng := engine.New() // no rules
	a, b := pipePair(t)
	c := chaosnet.WrapConn(a, eng)

	go func() { _, _ = c.Write([]byte("hi")) }()
	buf := make([]byte, 2)
	if _, err := io.ReadFull(b, buf); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	if string(buf) != "hi" {
		t.Fatalf("got %q, want \"hi\"", buf)
	}
	_ = context.Background()
}

func TestWrapListenerAutoWrapsAcceptedConns(t *testing.T) {
	sentinel := errors.New("server down")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("srv"))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	wln := chaosnet.WrapListener(ln, eng)

	accepted := make(chan net.Conn, 1)
	go func() {
		c, aerr := wln.Accept()
		if aerr == nil {
			accepted <- c
		}
	}()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	srvConn := <-accepted
	t.Cleanup(func() { _ = srvConn.Close() })
	// The accepted conn is chaos-wrapped: its Read faults.
	if _, err := srvConn.Read(make([]byte, 4)); !errors.Is(err, sentinel) {
		t.Fatalf("accepted conn Read = %v, want sentinel (auto-wrapped)", err)
	}
}

func TestWrapDialerFaultsAtDial(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.MatchName("198.51.100.1:9"), // the dial target
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("dial-drop"))

	called := false
	inner := func(context.Context, string, string) (net.Conn, error) {
		called = true
		return nil, nil
	}
	d := chaosnet.WrapDialer(eng, inner)
	conn, err := d.DialContext(context.Background(), "tcp", "198.51.100.1:9")

	if conn != nil {
		t.Fatalf("conn = %v, want nil", conn)
	}
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("err = %T %v, want *net.OpError", err, err)
	}
	if called {
		t.Fatal("inner dialer should not run when ConnDrop fires at dial")
	}
}

func TestWrapDialerReturnsWrappedConn(t *testing.T) {
	// No rules: dial succeeds and the returned conn is chaos-wrapped (a later
	// rule would fault its Read). Use net.Pipe for the inner conn.
	eng := engine.New()
	a, _ := pipePair(t)
	inner := func(context.Context, string, string) (net.Conn, error) { return a, nil }
	d := chaosnet.WrapDialer(eng, inner)

	conn, err := d.DialContext(context.Background(), "tcp", "x:1")
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	// Install a rule, then the dialed conn's Read should fault.
	sentinel := errors.New("post-dial drop")
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.MatchAttr("network", "pipe"),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("post"))
	if _, err := conn.Read(make([]byte, 4)); !errors.Is(err, sentinel) {
		t.Fatalf("dialed conn Read = %v, want sentinel (conn was wrapped)", err)
	}
}

func TestReadDisconnectMapsToEOF(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.Always(),
		engine.WithFault(fault.Disconnect()),
	).Named("close"))

	a, _ := pipePair(t)
	c := chaosnet.WrapConn(a, eng)
	_, err := c.Read(make([]byte, 8))
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("err = %T %v, want *net.OpError", err, err)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err chain = %v, want io.EOF inside", err)
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal("graceful Disconnect must not look like a hard reset")
	}
	if errors.Is(err, fault.ErrDisconnect) {
		t.Fatal("raw chaotic sentinel must not leak to the caller")
	}
}
