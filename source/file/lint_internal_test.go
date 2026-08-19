package file

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// hazardYAML is a rule LintSpecs rates SeverityHigh: a terminal fault with no
// kind and no name scope.
const hazardYAML = `
meta:
  version: 1
rules:
  - name: wipeout
    faults:
      - type: panic
        message: boom
`

func TestParseLintOffIsDefault(t *testing.T) {
	rs, err := Parse([]byte(hazardYAML))
	if err != nil {
		t.Fatalf("Parse without options rejected a hazard: %v", err)
	}
	if rs.Len() != 1 {
		t.Fatalf("got %d rules, want 1", rs.Len())
	}
}

func TestParseLintRejectFailsOnHazard(t *testing.T) {
	_, err := Parse([]byte(hazardYAML), WithLint(engine.LintReject))
	if !errors.Is(err, engine.ErrLintRejected) {
		t.Fatalf("err = %v, want ErrLintRejected", err)
	}
}

func TestParseLintWarnLogsAndAllows(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	rs, err := Parse([]byte(hazardYAML), WithLint(engine.LintWarn), withLogger(logger))
	if err != nil {
		t.Fatalf("LintWarn rejected the load: %v", err)
	}
	if rs.Len() != 1 {
		t.Fatalf("got %d rules, want 1", rs.Len())
	}
	if !strings.Contains(buf.String(), "wipeout") {
		t.Errorf("log %q does not name the offending rule", buf.String())
	}
}
