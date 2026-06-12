// Package prometheus provides a chaotic engine Observer that exports
// Prometheus counters for chaos rule and skips.
package prometheus

import (
	"context"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/prometheus/client_golang/prometheus"
)

// maxLabelLen bounds label cardinality by truncating long values, mirroring the
// discipline a high-traffic chaos target needs to keep Prometheus series bounded.
const maxLabelLen = 64

// Observer exports chaotic_rule_fires_total{rule},
// chaotic_rule_skips_total{rule,reason}, and the chaotic_fault_latency_seconds
// histogram{rule} of injected sleep durations.
type Observer struct {
	fires   *prometheus.CounterVec
	skips   *prometheus.CounterVec
	latency *prometheus.HistogramVec
}

// New registers the metrics with reg and returns the Observer. reg is a
// prometheus.Registerer so callers can pass a WrapRegistererWith(...)-wrapped
// registerer (constant labels / prefixes), per "accept interfaces".
func New(reg prometheus.Registerer) *Observer {
	o := &Observer{
		fires: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "chaotic_rule_fires_total",
			Help: "Number of times a chaos rule fired.",
		}, []string{"rule"}),
		skips: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "chaotic_rule_skips_total",
			Help: "Number of times a chaos rule was skipped.",
		}, []string{"rule", "reason"}),
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "chaotic_fault_latency_seconds",
			Help:    "Injected latency-fault sleep durations, in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"rule"}),
	}
	reg.MustRegister(o.fires, o.skips, o.latency)
	return o
}

// RuleFired adds 1 to fired rule counter.
func (o *Observer) RuleFired(ruleName string, _ engine.Op, _ engine.Action) {
	o.fires.WithLabelValues(truncateLabel(ruleName)).Inc()
}

// RuleSkipped adds 1 to skipped rule counter.
func (o *Observer) RuleSkipped(ruleName string, _ engine.Op, reason string) {
	o.skips.WithLabelValues(truncateLabel(ruleName), reason).Inc()
}

// FaultInjected records the injected sleep duration of a latency or jittered
// fault into the chaotic_fault_latency_seconds histogram.
func (o *Observer) FaultInjected(_ context.Context, ev engine.FaultEvent) {
	o.latency.WithLabelValues(truncateLabel(ev.Rule)).Observe(ev.Latency.Seconds())
}

// truncateLabel caps a label value at maxLabelLen bytes to bound cardinality.
func truncateLabel(s string) string {
	if len(s) > maxLabelLen {
		return s[:maxLabelLen]
	}
	return s
}

var (
	_ engine.Observer     = (*Observer)(nil)
	_ engine.RichObserver = (*Observer)(nil)
)
