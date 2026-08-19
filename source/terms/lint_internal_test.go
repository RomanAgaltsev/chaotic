package terms

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// hazard is a terms string LintSpecs rates SeverityHigh: a terminal fault with
// no kind and no name scope.
const hazard = `wipeout: panic("boom")`

func TestParseLintOffIsDefault(t *testing.T) {
	specs, err := Parse(hazard)
	if err != nil {
		t.Fatalf("Parse without options rejected a hazard: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want 1", len(specs))
	}
}

func TestParseLintRejectFailsOnHazard(t *testing.T) {
	_, err := Parse(hazard, WithLint(engine.LintReject))
	if !errors.Is(err, engine.ErrLintRejected) {
		t.Fatalf("err = %v, want ErrLintRejected", err)
	}
}

func TestParseLintWarnLogsAndAllows(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	specs, err := Parse(hazard, WithLint(engine.LintWarn), withLogger(logger))
	if err != nil {
		t.Fatalf("LintWarn rejected the load: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want 1", len(specs))
	}
	if !strings.Contains(buf.String(), "wipeout") {
		t.Errorf("log %q does not name the offending rule", buf.String())
	}
}

func TestCompileLintRejectFailsOnHazard(t *testing.T) {
	_, err := Compile(hazard, WithLint(engine.LintReject))
	if !errors.Is(err, engine.ErrLintRejected) {
		t.Fatalf("err = %v, want ErrLintRejected", err)
	}
}

func TestCompileLintsExactlyOnce(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	if _, err := Compile(hazard, WithLint(engine.LintWarn), withLogger(logger)); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if n := strings.Count(buf.String(), "wipeout"); n != 1 {
		t.Errorf("rule logged %d times, want 1 (Compile must not lint on top of Parse)", n)
	}
}
