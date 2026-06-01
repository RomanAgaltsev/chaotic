// Package slog provides a chaotic engine Observer that emits structured logs
// via log/slog. Construct with New and pass to engine.WithObserver.
package slog

import (
	"context"
	"log/slog"

	"github.com/ag4r/chaotic/engine"
)

// Observer logs chaos rule fires (info) and skips (debug) to a slog.Logger.
type Observer struct {
	logger *slog.Logger
}

// New returns an Observer logging to logger, or slog.Default() if logger is nil.
func New(logger *slog.Logger) *Observer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Observer{
		logger: logger,
	}
}

// RuleFired adds 1 to fired rule counter.
func (o *Observer) RuleFired(ruleName string, op engine.Op, _ engine.Action) {
	o.logger.LogAttrs(context.Background(), slog.LevelInfo, "chaos rule fired",
		slog.String("rule", ruleName),
		slog.Int("kind", int(op.Kind)),
		slog.String("op_name", op.Name),
		slog.String("method", op.Method),
	)
}

// RuleSkipped adds 1 to skipped rule counter.
func (o *Observer) RuleSkipped(ruleName string, op engine.Op, reason string) {
	o.logger.LogAttrs(context.Background(), slog.LevelDebug, "chaos rule skipped",
		slog.String("rule", ruleName),
		slog.String("reason", reason),
		slog.Int("kind", int(op.Kind)),
		slog.String("op_name", op.Name),
	)
}

var _ engine.Observer = (*Observer)(nil)
