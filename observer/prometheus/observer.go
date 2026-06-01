// Package prometheus provides a chaotic engine Observer that exports
// Prometheus counters for chaos rule and skips.
package prometheus

import (
	"github.com/ag4r/chaotic/engine"
	"github.com/prometheus/client_golang/prometheus"
)

// Observer exports chaotic_rule_fires_total{rule} and
// chaotic_rule_skips_total{rule,reason}.
type Observer struct {
	fires *prometheus.CounterVec
	skips *prometheus.CounterVec
}

// New registers the metrics with reg and returns the Observer.
func New(reg *prometheus.Registry) *Observer {
	o := &Observer{
		fires: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "chaotic_rule_fires_total",
			Help: "Number of times a chaos rule fired.",
		}, []string{"rule"}),
		skips: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "chaotic_rule_skips_total",
			Help: "Number of times a chaos rule was skipped.",
		}, []string{"rule"}),
	}
	reg.MustRegister(o.fires, o.skips)
	return o
}

// RuleFired adds 1 to fired rule counter.
func (o *Observer) RuleFired(ruleName string, _ engine.Op, _ engine.Action) {
	o.fires.WithLabelValues(ruleName).Inc()
}

// RuleSkipped adds 1 to skipped rule counter.
func (o *Observer) RuleSkipped(ruleName string, _ engine.Op, reason string) {
	o.skips.WithLabelValues(ruleName, reason).Inc()
}

var _ engine.Observer = (*Observer)(nil)
