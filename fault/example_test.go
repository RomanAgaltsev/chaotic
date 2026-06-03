package fault_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ag4r/chaotic/fault"
)

func ExampleError() {
	f := fault.Error(errors.New("upstream unavailable"))
	fmt.Println(f.Apply(context.Background()))
	// Output: upstream unavailable
}

func ExampleConnDrop() {
	// ConnDrop returns a sentinel each adapter maps to its native
	// connection-drop error (driver.ErrBadConn, codes.Unavailable, ...).
	f := fault.ConnDrop()
	err := f.Apply(context.Background())
	fmt.Println(errors.Is(err, fault.ErrConnDrop))
	// Output: true
}

func ExampleLatency_contextCancellation() {
	// A canceled context makes a latency fault return immediately with the
	// context error instead of sleeping.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := fault.Latency(time.Hour)
	fmt.Println(f.Apply(ctx))
	// Output: context canceled
}

func ExamplePanic() {
	f := fault.Panic("boom")
	defer func() {
		fmt.Println("recovered:", recover())
	}()
	_ = f.Apply(context.Background())
	// Output: recovered: boom
}

func ExampleJitteredSeed() {
	// JitteredSeed draws sleep durations from a seeded source, so a chaos test
	// replays the same delays across runs. No Output: the delays are time-based
	// and intentionally not printed.
	f := fault.JitteredSeed(10*time.Millisecond, 50*time.Millisecond, 42)
	_ = f.Apply(context.Background())
}
