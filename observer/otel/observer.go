// Package otel provides a chaotic engine Observer that records OpenTelemetry
// metric counters for chaos rule fires and skips.
package otel

import (
	"context"

	"github.com/ag4r/chaotic/engine"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Observer records chaotic.rule.fires and chaotic.rule.skips via an OTel meter.
type Observer struct {
	fires metric.Int64Counter
	skips metric.Int64Counter
}

// New creates the counters on meter and returns the Observer.
func New(meter metric.Meter) (*Observer, error) {
	fires, err := meter.Int64Counter("chaotic.rule.fires",
		metric.WithDescription("Number of times a chaos rule fired."))
	if err != nil {
		return nil, err
	}
	skips, err := meter.Int64Counter("chaotic.rule.skips",
		metric.WithDescription("Number of times a chaos rule skipped."))
	if err != nil {
		return nil, err
	}
	return &Observer{
		fires: fires,
		skips: skips,
	}, nil
}

// RuleFired adds 1 to fired rule counter.
func (o *Observer) RuleFired(ruleName string, op engine.Op, _ engine.Action) {
	o.fires.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String("rule", ruleName),
		attribute.Int("chaotic.kind", int(op.Kind)),
	))
}

// RuleSkipped adds 1 to skipped rule counter.
func (o *Observer) RuleSkipped(ruleName string, op engine.Op, reason string) {
	o.skips.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String("rule", ruleName),
		attribute.String("reason", reason),
		attribute.Int("chaotic.kind", int(op.Kind)),
	))
}

// FaultInjected records the injected fault as an event on the span active in
// ctx, with the fault kind, op kind, and (for latency/jittered faults) the
// injected sleep in seconds. It is a no-op when no recording span is active.
func (o *Observer) FaultInjected(ctx context.Context, ev engine.FaultEvent) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	span.AddEvent("chaotic.fault_injected", trace.WithAttributes(
		attribute.String("rule", ev.Rule),
		attribute.Int("chaotic.fault_kind", int(ev.FaultKind)),
		attribute.Int("chaotic.kind", int(ev.Op.Kind)),
		attribute.Float64("chaotic.latency_seconds", ev.Latency.Seconds()),
	))
}

var (
	_ engine.Observer     = (*Observer)(nil)
	_ engine.RichObserver = (*Observer)(nil)
)
