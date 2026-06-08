//go:build !chaos_off

package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// A WithMaxConcurrent slot must be released after each command, or chaos silently
// stops once the cap is reached. fault.Latency(0) returns nil from Before, so the
// slot is freed only by InsertOne running After.
func TestInsertOneReleasesMaxConcurrentSlot(t *testing.T) {
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpMongo),
		engine.Always(),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))

	c := newColl(&fakeColl{}, eng)
	ctx := context.Background()
	for range 3 {
		if _, err := c.InsertOne(ctx, bson.D{{Key: "x", Value: 1}}); err != nil {
			t.Fatalf("InsertOne err = %v", err)
		}
	}
	if got := eng.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential inserts; the max-concurrent slot is leaking", got)
	}
}
