package fault

import (
	"context"
	"fmt"
)

// StreamMode discriminates the three stream-shaping faults inside a
// StreamFaultError.
type StreamMode int

// Stream-shaping modes.
const (
	StreamSlowRead StreamMode = iota
	StreamSlowWrite
	StreamTruncate
)

// StreamFaultError is the sentinel the stream-shaping faults return. adapter/io
// detects it via errors.As and shapes the data stream instead of aborting the
// call. The fault package stays free of io-wrapping code: this carries only the
// mode and its parameters.
type StreamFaultError struct {
	Mode  StreamMode
	Rate  int // bytes/sec for the slow modes; 0 means "block until ctx" (never-ends)
	Limit int // byte cap for StreamTruncate
}

func (e *StreamFaultError) Error() string {
	switch e.Mode {
	case StreamTruncate:
		return fmt.Sprintf("chaotic: stream fault (truncate %d B)", e.Limit)
	case StreamSlowWrite:
		return fmt.Sprintf("chaotic: stream fault (slow write %d B/s)", e.Rate)
	default:
		return fmt.Sprintf("chaotic: stream fault (slow read %d B/s)", e.Rate)
	}
}

// SlowReader rate-limits reads from a chaosio-wrapped reader to rate bytes/sec.
// rate == 0 blocks until the context is done (models a body that never ends).
func SlowReader(rate int) Fault { return streamFault{mode: StreamSlowRead, rate: rate} }

// SlowWriter is the symmetric write-side rate limit.
func SlowWriter(rate int) Fault { return streamFault{mode: StreamSlowWrite, rate: rate} }

// Truncate cuts a chaosio-wrapped stream off after n bytes: a reader returns
// io.EOF past n, a writer returns io.ErrShortWrite past n.
func Truncate(n int) Fault { return streamFault{mode: StreamTruncate, limit: n} }

type streamFault struct {
	mode  StreamMode
	rate  int
	limit int
}

func (s streamFault) Apply(context.Context) error {
	return &StreamFaultError{Mode: s.mode, Rate: s.rate, Limit: s.limit}
}

func (s streamFault) Kind() Kind {
	switch s.mode {
	case StreamSlowWrite:
		return KindSlowWriter
	case StreamTruncate:
		return KindTruncate
	default:
		return KindSlowReader
	}
}
