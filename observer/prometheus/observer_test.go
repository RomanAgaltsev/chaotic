package prometheus_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

func histogramCount(t *testing.T, reg *prometheus.Registry, name, rule string) uint64 {
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
					return m.GetHistogram().GetSampleCount()
				}
			}
		}
	}
	return 0
}

func TestObserverRecordsFaultLatency(t *testing.T) {
	reg := prometheus.NewRegistry()
	obs := chaosprom.New(reg)
	e := engine.New(engine.WithObserver(obs)).
		AddRule(engine.NewRule(engine.WithFault(fault.Latency(5 * time.Millisecond))).Named("slow"))
	ctx := context.Background()
	a := e.Eval(ctx, engine.Op{})
	if err := a.Before(ctx); err != nil {
		t.Fatalf("Before: %v", err)
	}
	if got := histogramCount(t, reg, "chaotic_fault_latency_seconds", "slow"); got != 1 {
		t.Fatalf("latency histogram sample count = %d, want 1", got)
	}
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

func TestObserverCountsSkips(t *testing.T) {
	reg := prometheus.NewRegistry()
	e := engine.New(engine.WithObserver(chaosprom.New(reg))).
		AddRule(engine.NewRule(
			engine.MatchKind(engine.OpHTTPClient),
			engine.Times(1),
			engine.WithFault(fault.Latency(0)),
		).Named("once"))
	ctx := context.Background()
	e.Eval(ctx, engine.Op{Kind: engine.OpHTTPClient}) // fires (Times(1))
	e.Eval(ctx, engine.Op{Kind: engine.OpHTTPClient}) // skipped: counter exhausted
	if got := counterValue(t, reg, "chaotic_rule_skips_total", "once"); got != 1 {
		t.Fatalf("skips = %v, want 1", got)
	}
}
