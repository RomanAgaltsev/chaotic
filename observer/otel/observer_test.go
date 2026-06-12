package otel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
	chaosotel "github.com/RomanAgaltsev/chaotic/observer/otel"
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

func TestObserverRecordsFaultSpanEvent(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracer := tp.Tracer("test")

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	obs, err := chaosotel.New(mp.Meter("test"))
	if err != nil {
		t.Fatal(err)
	}
	e := engine.New(engine.WithObserver(obs)).
		AddRule(engine.NewRule(engine.WithFault(fault.Latency(3 * time.Millisecond))).Named("slow"))

	ctx, span := tracer.Start(context.Background(), "parent")
	a := e.Eval(ctx, engine.Op{Kind: engine.OpHTTPClient})
	if err := a.Before(ctx); err != nil {
		t.Fatalf("Before: %v", err)
	}
	span.End()

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("recorded spans = %d, want 1", len(spans))
	}
	var found bool
	for _, ev := range spans[0].Events() {
		if ev.Name == "chaotic.fault_injected" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no chaotic.fault_injected span event recorded; events = %+v", spans[0].Events())
	}
}
