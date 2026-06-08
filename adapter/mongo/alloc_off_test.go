//go:build chaos_off

package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestZeroAllocUnderChaosOff(t *testing.T) {
	c := &Collection{coll: &fakeColl{}}
	ctx := context.Background()
	// Box the document into any once, outside the measured closure: the same
	// conversion happens when calling the real driver, so it is not wrapper
	// overhead. This isolates the measurement to the passthrough itself.
	var doc any = bson.D{{Key: "x", Value: 1}}
	avg := testing.AllocsPerRun(100, func() {
		_, _ = c.InsertOne(ctx, doc)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
