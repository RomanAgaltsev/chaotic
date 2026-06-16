package pgx

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func baseConfig(t *testing.T) *pgxpool.Config {
	t.Helper()
	// A parseable DSN; we never actually connect.
	cfg, err := pgxpool.ParseConfig("postgres://u:p@localhost:5432/db")
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	return cfg
}

func TestInstrumentPoolConfig_ReturnsSameConfigType(t *testing.T) {
	cfg := baseConfig(t)
	eng := engine.New()

	got := InstrumentPoolConfig(cfg, eng)

	// Genuinely the same *pgxpool.Config (zero consumer type change).
	if got != cfg {
		t.Fatal("InstrumentPoolConfig must return the same *pgxpool.Config it was given")
	}
}

func TestInstrumentPoolConfig_WiresDialFuncWithChaos(t *testing.T) {
	cfg := baseConfig(t)
	// Inner dialer we can detect was chained, returning a fake conn.
	innerCalled := false
	cfg.ConnConfig.DialFunc = func(_ context.Context, _, _ string) (net.Conn, error) {
		innerCalled = true
		c, _ := net.Pipe()
		return c, nil
	}

	// Engine that drops the dial once.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("dial refused"))),
	).Named("dial-flap"))

	InstrumentPoolConfig(cfg, eng, WithoutQueryLatency())

	_, err := cfg.ConnConfig.DialFunc(context.Background(), "tcp", "localhost:5432")
	if err == nil {
		t.Fatal("expected chaos to fault the dial, got nil")
	}
	if innerCalled {
		t.Fatal("inner dialer should not run when chaos faults the dial")
	}
}

func TestInstrumentPoolConfig_ChainsExistingDialFunc(t *testing.T) {
	cfg := baseConfig(t)
	innerCalled := false
	cfg.ConnConfig.DialFunc = func(_ context.Context, _, _ string) (net.Conn, error) {
		innerCalled = true
		c, _ := net.Pipe()
		return c, nil
	}
	eng := engine.New() // disabled: no rules -> pass-through

	InstrumentPoolConfig(cfg, eng, WithoutQueryLatency())

	if _, err := cfg.ConnConfig.DialFunc(context.Background(), "tcp", "x:1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !innerCalled {
		t.Fatal("existing DialFunc must be chained, not discarded")
	}
}

func TestInstrumentPoolConfig_WithoutDialFaults_LeavesDialFunc(t *testing.T) {
	cfg := baseConfig(t)
	sentinel := func(_ context.Context, _, _ string) (net.Conn, error) {
		return nil, errors.New("sentinel")
	}
	cfg.ConnConfig.DialFunc = sentinel
	eng := engine.New()

	InstrumentPoolConfig(cfg, eng, WithoutDialFaults(), WithoutQueryLatency())

	_, err := cfg.ConnConfig.DialFunc(context.Background(), "tcp", "x:1")
	if err == nil || err.Error() != "sentinel" {
		t.Fatalf("DialFunc should be untouched, got err=%v", err)
	}
}

func TestInstrumentPoolConfig_NilGuards(t *testing.T) {
	t.Run("nil config panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic on nil config")
			}
		}()
		InstrumentPoolConfig(nil, engine.New())
	})
	t.Run("nil engine panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic on nil engine")
			}
		}()
		InstrumentPoolConfig(baseConfig(t), nil)
	})
}

// referenced by Task 3 tests; declared here so the file compiles incrementally.
var _ = time.Millisecond
