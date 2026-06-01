package slog_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
	chaosslog "github.com/ag4r/chaotic/observer/slog"
)

func TestObserverLogsRuleFired(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	e := engine.New(engine.WithObserver(chaosslog.New(logger))).
		AddRule(engine.NewRule(engine.WithFault(fault.Error(errors.New("x")))).Named("boom"))
	e.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient, Name: "/x"})
	out := buf.String()
	if !strings.Contains(out, `"rule":"boom"`) || !strings.Contains(out, "chaos rule fired") {
		t.Fatalf("log missing expected fields: %s", out)
	}
}
