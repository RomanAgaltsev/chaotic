package pgx

import (
	"errors"
	"io"
	"net"
	"testing"

	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestTranslateNil(t *testing.T) {
	if got := translate(nil); got != nil {
		t.Fatalf("translate(nil) = %v, want nil", got)
	}
}

func TestTranslatePassesArbitraryError(t *testing.T) {
	err := errors.New("boom")
	if got := translate(err); got != err {
		t.Fatalf("translate(boom) = %v, want same error", got)
	}
}

func TestTranslateConnDropToNetOpError(t *testing.T) {
	got := translate(fault.ErrConnDrop)
	var opErr *net.OpError
	if !errors.As(got, &opErr) {
		t.Fatalf("translate(ErrConnDrop) = %v (%T), want *net.OpError", got, got)
	}
	if opErr.Op != "read" {
		t.Errorf("OpError.Op = %q, want %q", opErr.Op, "read")
	}
	if opErr.Net != "tcp" {
		t.Errorf("OpError.Net = %q, want %q", opErr.Net, "tcp")
	}
	if !errors.Is(opErr.Err, io.ErrUnexpectedEOF) {
		t.Errorf("OpError.Err = %v, want errors.Is(io.ErrUnexpectedEOF) == true", opErr.Err)
	}
}
