package prometheus_test

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
	chaosprom "github.com/ag4r/chaotic/observer/prometheus"
)

func counterValue(t *testing.T, reg *prometheus.Registry, name, rule string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "rule" && l.GetValue() == rule {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

func TestObserverCountsFires(t *testing.T) {
	reg := prometheus.NewRegistry()
	e := engine.New(engine.WithObserver(chaosprom.New(reg))).
		AddRule(engine.NewRule(engine.WithFault(fault.Error(errors.New("x")))).Named("boom"))
	e.Eval(context.Background(), engine.Op{})
	e.Eval(context.Background(), engine.Op{})
	if got := counterValue(t, reg, "chaotic_rule_fires_total", "boom"); got != 2 {
		t.Fatalf("fires = %v, want 2", got)
	}
}
