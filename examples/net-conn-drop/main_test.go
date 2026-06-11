package main

import (
	"net"
	"testing"

	chaosnet "github.com/ag4r/chaotic/adapter/net"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestReadRetrySurvivesDrop(t *testing.T) {
	a, b := net.Pipe()
	t.Cleanup(func() { _ = a.Close(); _ = b.Close() })

	eng := engine.New()
	c := chaosnet.WrapConn(a, eng)

	// Drop the first two reads, then let them through.
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.Times(2),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	go func() {
		// The writer keeps the data available; the first two reads are faulted
		// before touching the pipe, so this write is consumed by the 3rd read.
		_, _ = b.Write([]byte("ok!!"))
	}()

	buf := make([]byte, 4)
	n, err := ReadWithRetry(c, buf, 5)
	if err != nil {
		t.Fatalf("ReadWithRetry failed despite retries: %v", err)
	}
	if n == 0 {
		t.Fatal("read 0 bytes")
	}
	if got := eng.Hits("drop"); got != 2 {
		t.Fatalf("drop fired %d times, want 2", got)
	}
}
