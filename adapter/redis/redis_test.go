//go:build !chaos_off

package redis_test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	goredis "github.com/redis/go-redis/v9"

	chaosredis "github.com/RomanAgaltsev/chaotic/adapter/redis"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// nextOK is a ProcessHook terminator that succeeds without touching the network.
func nextOK(_ context.Context, _ goredis.Cmder) error {
	return nil
}

func TestProcessHookInjectsError(t *testing.T) {
	sentinel := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("err"))

	ph := chaosredis.NewHook(eng).ProcessHook(nextOK)
	cmd := goredis.NewCmd(context.Background(), "get", "k")
	err := ph(context.Background(), cmd)

	if !errors.Is(err, sentinel) {
		t.Fatalf("returned err = %v, want sentinel", err)
	}
	if !errors.Is(cmd.Err(), sentinel) {
		t.Fatalf("cmd.Err() = %v, want sentinel set on the command", cmd.Err())
	}
}

func TestProcessHookConnDropMapsToNetOpError(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	ph := chaosredis.NewHook(eng).ProcessHook(nextOK)
	cmd := goredis.NewCmd(context.Background(), "get", "k")
	err := ph(context.Background(), cmd)

	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("err = %T %v, want *net.OpError", err, err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err chain = %v, want io.ErrUnexpectedEOF inside", err)
	}
}

func TestProcessHookPassesThroughWhenNoRule(t *testing.T) {
	eng := engine.New() // no rules
	called := false
	next := func(context.Context, goredis.Cmder) error { called = true; return nil }

	ph := chaosredis.NewHook(eng).ProcessHook(next)
	cmd := goredis.NewCmd(context.Background(), "get", "k")
	if err := ph(context.Background(), cmd); err != nil {
		t.Fatalf("err = %v, want nil passthrough", err)
	}
	if !called {
		t.Fatal("next was not called on the no-rule path")
	}
}

func TestDialHookConnDrop(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.MatchName("DIAL"),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("dial-drop"))

	nextDial := func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("next dialer should not be reached when ConnDrop fires")
		return nil, nil
	}
	dh := chaosredis.NewHook(eng).DialHook(nextDial)
	conn, err := dh(context.Background(), "tcp", "localhost:6379")

	if conn != nil {
		t.Fatalf("conn = %v, want nil", conn)
	}
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("err = %T %v, want *net.OpError", err, err)
	}
}

func TestProcessPipelineHookFaultsOnceAtOpen(t *testing.T) {
	sentinel := errors.New("pipeline down")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("pipe-err"))

	nextPipe := func(context.Context, []goredis.Cmder) error {
		t.Fatal("next should not run when the pipeline faults at open")
		return nil
	}
	pph := chaosredis.NewHook(eng).ProcessPipelineHook(nextPipe)

	ctx := context.Background()
	cmds := []goredis.Cmder{
		goredis.NewCmd(ctx, "get", "a"),
		goredis.NewCmd(ctx, "get", "b"),
	}
	err := pph(ctx, cmds)

	if !errors.Is(err, sentinel) {
		t.Fatalf("returned err = %v, want sentinel", err)
	}
	// The fault fired ONCE at open: the rule hit exactly once, not per command.
	if got := eng.Hits("pipe-err"); got != 1 {
		t.Fatalf("rule fired %d times, want 1 (once at pipeline open)", got)
	}
	// Every command in the batch carries the error.
	for i, c := range cmds {
		if !errors.Is(c.Err(), sentinel) {
			t.Fatalf("cmds[%d].Err() = %v, want sentinel", i, c.Err())
		}
	}
}
