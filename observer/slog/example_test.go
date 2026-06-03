package slog_test

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
	chaosslog "github.com/ag4r/chaotic/observer/slog"
)

func ExampleNew() {
	// Strip the time attribute so the logged line is deterministic.
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})
	eng := engine.New(engine.WithObserver(chaosslog.New(slog.New(h)))).
		AddRule(engine.NewRule(
			engine.MatchKind(engine.OpHTTPClient),
			engine.WithFault(fault.Error(errors.New("boom"))),
		).Named("demo"))

	// Evaluating a matching op fires the rule, which the observer logs.
	_ = eng.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient, Name: "/users", Method: "GET"})
	// Output:
	// level=INFO msg="chaos rule fired" rule=demo kind=1 op_name=/users method=GET
}
