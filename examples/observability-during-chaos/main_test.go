package main

import (
	"slices"
	"testing"
)

func TestObserverSeesChaosFire(t *testing.T) {
	rec := &recorder{}
	_ = run(rec) // the GET returns the injected error; we only care about the observer
	if !slices.Contains(rec.Fired(), "http-fail") {
		t.Fatalf("observer did not record the fire; saw %v", rec.Fired())
	}
}
