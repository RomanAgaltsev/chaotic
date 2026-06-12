package fault

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLatencyAppliesDuration(t *testing.T) {
	f := Latency(40 * time.Millisecond)
	start := time.Now()
	if err := f.Apply(context.Background()); err != nil {
		t.Fatalf("Apply returned %v, want nil", err)
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("sleep too short: %v", elapsed)
	}
}

func TestLatencyHonorsContextCancellation(t *testing.T) {
	f := Latency(10 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	err := f.Apply(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Apply returned %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("Apply did not exit early: %v", elapsed)
	}
}

func TestLatencyZeroIsImmediate(t *testing.T) {
	if err := Latency(0).Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestJitteredStaysInRange(t *testing.T) {
	min, max := 10*time.Millisecond, 30*time.Millisecond
	f := Jittered(min, max)
	for i := 0; i < 10; i++ {
		start := time.Now()
		if err := f.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
		elapsed := time.Since(start)
		if elapsed < min/2 || elapsed > max*3 {
			t.Fatalf("iteration %d: elapsed %v outside expected range [%v, %v]", i, elapsed, min, max)
		}
	}
}

func TestJitteredHonorsContextCancellation(t *testing.T) {
	f := Jittered(5*time.Second, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	err := f.Apply(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Apply returned %v, want context.Canceled", err)
	}
}

func TestErrorReturnsWrapped(t *testing.T) {
	sentinel := errors.New("boom")
	if err := Error(sentinel).Apply(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("Apply returned %v, want errors.Is(%v)", err, sentinel)
	}
}

func TestErrorNilReturnsNil(t *testing.T) {
	if err := Error(nil).Apply(context.Background()); err != nil {
		t.Fatalf("Apply returned %v, want nil", err)
	}
}

func TestPanicPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got none")
		}
		if r != "kaboom" {
			t.Fatalf("recovered %v, want \"kaboom\"", r)
		}
	}()
	_ = Panic("kaboom").Apply(context.Background())
}

func TestConnDropReturnsSentinel(t *testing.T) {
	err := ConnDrop().Apply(context.Background())
	if !errors.Is(err, ErrConnDrop) {
		t.Fatalf("Apply returned %v, want errors.Is(ErrConnDrop) == true", err)
	}
}

func TestJitteredSeedIsReproducible(t *testing.T) {
	min, max := 10*time.Millisecond, 100*time.Millisecond
	// Two faults with the same seed must choose identical durations. Assert via
	// elapsed time being equal across paired runs (or expose a test seam).
	a := JitteredSeed(min, max, 42)
	b := JitteredSeed(min, max, 42)
	for range 5 {
		ta := timeApply(t, a)
		tb := timeApply(t, b)
		// Same seed, same call index -> same draw. timeApply reads the intended
		// draw through a seam, so equality is exact (no scheduler slop).
		if ta != tb {
			t.Fatalf("draw mismatch: a=%v b=%v", ta, tb)
		}
		if ta < min || ta > max {
			t.Fatalf("draw %v out of range [%v, %v]", ta, min, max)
		}
	}
}

// timeApply returns the duration f would sleep for. For seeded jitter it reads
// the intended draw through the seam (deterministic, no flakiness); for any
// other fault it measures the wall-clock time Apply spends.
func timeApply(t *testing.T, f Fault) time.Duration {
	t.Helper()
	if sj, ok := f.(*seededJitter); ok {
		return sj.draw()
	}
	start := time.Now()
	if err := f.Apply(context.Background()); err != nil {
		t.Fatalf("Apply returned %v, want nil", err)
	}
	return time.Since(start)
}

func TestDisconnectReturnsSentinel(t *testing.T) {
	err := Disconnect().Apply(context.Background())
	if !errors.Is(err, ErrDisconnect) {
		t.Fatalf("Apply returned %v, want errors.Is(ErrDisconnect) == true", err)
	}
	if errors.Is(err, ErrConnDrop) {
		t.Fatal("Disconnect must not alias ErrConnDrop")
	}
	if KindOf(Disconnect()) != KindDisconnect {
		t.Fatalf("KindOf(Disconnect()) = %v, want KindDisconnect", KindOf(Disconnect()))
	}
}
