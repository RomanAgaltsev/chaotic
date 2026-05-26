package quick_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ag4r/chaotic/chaostest"
	"github.com/ag4r/chaotic/chaostest/quick"
	"github.com/ag4r/chaotic/engine"
)

func TestFailFirstFiresExactlyN(t *testing.T) {
	e := chaostest.New(t)
	target := errors.New("boom")
	quick.FailFirst(t, e, 2, engine.OpHTTPClient, target)
	for i := range 5 {
		a := e.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient})
		err := a.Before(context.Background())
		if i < 2 {
			if !errors.Is(err, target) {
				t.Fatalf("iteration %d: err=%v, want target", i, err)
			}
		} else {
			if err != nil {
				t.Fatalf("iteration %d: err=%v, want nil", i, err)
			}
		}
	}
}

func TestSlowAlwaysAddsLatency(t *testing.T) {
	e := chaostest.New(t)
	quick.SlowAlways(t, e, engine.OpSQL, 40*time.Millisecond)
	for i := range 3 {
		start := time.Now()
		a := e.Eval(context.Background(), engine.Op{Kind: engine.OpSQL})
		if err := a.Before(context.Background()); err != nil {
			t.Fatal(err)
		}
		if time.Since(start) < 30*time.Millisecond {
			t.Fatalf("iteration %d: too fast", i)
		}
	}
}

func TestPanicOnceFiresOnce(t *testing.T) {
	e := chaostest.New(t)
	quick.PanicOnce(t, e, engine.OpGRPCClient, "kaboom")
	// First call: panics
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()
		a := e.Eval(context.Background(), engine.Op{Kind: engine.OpGRPCClient})
		_ = a.Before(context.Background())
	}()
	// Second call: no panic.
	a := e.Eval(context.Background(), engine.Op{Kind: engine.OpGRPCClient})
	if err := a.Before(context.Background()); err != nil {
		t.Fatalf("second call: %v", err)
	}
}
