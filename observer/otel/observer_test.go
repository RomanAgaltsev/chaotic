package otel_test

import (
	"context"
	"errors"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
	chaosotel "github.com/ag4r/chaotic/observer/otel"
)

func TestObserverRecordsFires(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	obs, err := chaosotel.New(mp.Meter("test"))
	if err != nil {
		t.Fatal(err)
	}
	e := engine.New(engine.WithObserver(obs)).
		AddRule(engine.NewRule(engine.WithFault(fault.Error(errors.New("x")))).Named("boom"))
	ctx := context.Background()
	e.Eval(ctx, engine.Op{})

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	var fires float64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "chaotic.rule.fires" {
				continue
			}
			if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
				for _, dp := range sum.DataPoints {
					fires += float64(dp.Value)
				}
			}
		}
	}
	if fires != 1 {
		t.Fatalf("fires = %v, want 1", fires)
	}
}
