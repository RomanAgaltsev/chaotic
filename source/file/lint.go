package file

import (
	"log/slog"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// Option configures Load, Parse and Watch.
type Option func(*config)

// config is the resolved option set. The zero value is the historical
// behavior: no linting, no logging.
type config struct {
	lint   engine.LintMode
	logger *slog.Logger
}

// WithLint sets the blast-radius lint policy applied to the parsed specs before
// any rule is built. engine.LintWarn logs findings and loads anyway;
// engine.LintReject fails the load when a finding is engine.SeverityHigh.
// Default: engine.LintOff.
func WithLint(mode engine.LintMode) Option {
	return func(c *config) { c.lint = mode }
}

// withLogger directs LintWarn findings to logger instead of slog.Default. It is
// unexported: Watch threads its own logger through it, and tests capture
// findings with it.
func withLogger(l *slog.Logger) Option {
	return func(c *config) { c.logger = l }
}

func newConfig(opts []Option) config {
	var c config
	for _, o := range opts {
		o(&c)
	}
	return c
}

// gate runs the lint policy over specs, logging findings under LintWarn. It
// returns a non-nil error only when the policy rejects the specs.
func (c config) gate(specs []engine.RuleSpec) error {
	rep, err := engine.LintGate(c.lint, specs)
	if err != nil {
		return err
	}
	c.logReport(rep)
	return nil
}

// logReport emits one warn record per finding. Nothing is logged when the
// report is empty, so LintOff and clean rule sets stay silent.
func (c config) logReport(rep engine.Report) {
	if len(rep.Findings) == 0 {
		return
	}
	logger := c.logger
	if logger == nil {
		logger = slog.Default()
	}
	for _, f := range rep.Findings {
		logger.Warn("chaotic: rule lint finding",
			"rule", f.Rule,
			"severity", f.Severity.String(),
			"message", f.Message,
		)
	}
}
