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

func ExampleHTTPStatus() {
	// HTTPStatus carries a status code on a sentinel the HTTP adapters render.
	err := fault.HTTPStatus(503, "overloaded").Apply(context.Background())
	var hse *fault.HTTPStatusError
	errors.As(err, &hse)
	fmt.Println(hse.StatusCode(), hse.Body)
	// Output: 503 overloaded
}

func ExampleHeaderStrip() {
	// HeaderStrip yields a sentinel describing a header deletion; the adapters
	// apply it to the headers flowing toward the code under test.
	err := fault.HeaderStrip("X-Trace-Id").Apply(context.Background())
	var hf *fault.HeaderFault
	errors.As(err, &hf)
	fmt.Println(hf.Strip, hf.Key)
	// Output: true X-Trace-Id
}

// ExampleClock shows that a Clock fault, once applied, sets the skew that
// fault.Skew (and engine.Now) read back from the context.
func ExampleClock() {
	ctx := fault.WithClock(context.Background())
	_ = fault.Clock(72 * time.Hour).Apply(ctx)
	fmt.Println(fault.Skew(ctx))
	// Output: 72h0m0s
}
