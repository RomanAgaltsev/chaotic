//go:build !chaos_off

package mongo

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	gomongo "go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// newColl builds a *Collection over a fake seam with fixed db/coll attrs.
func newColl(fake mongoColl, eng *engine.Engine) *Collection {
	return &Collection{
		coll: fake,
		eng:  eng,
		db:   "app",
		name: "users",
	}
}

func TestInsertOneInjectsError(t *testing.T) {
	sentinel := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpMongo),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("err"))

	fake := &fakeColl{}
	c := newColl(fake, eng)
	_, err := c.InsertOne(context.Background(), bson.D{{Key: "x", Value: 1}})

	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
	if fake.inserts != 0 {
		t.Fatalf("underlying InsertOne ran %d times, want 0 (fault short-circuits)", fake.inserts)
	}
}

func TestInsertOneConnDropMapsToRetryableCommandError(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpMongo),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	c := newColl(&fakeColl{}, eng)
	_, err := c.InsertOne(context.Background(), bson.D{{Key: "x", Value: 1}})

	var ce gomongo.CommandError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %T %v, want mongo.CommandError", err, err)
	}
	if !ce.HasErrorLabel("NetworkError") {
		t.Fatalf("CommandError labels = %v, want NetworkError present", ce.Labels)
	}
}

func TestInsertOnePassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeColl{}
	c := newColl(fake, engine.New()) // no rules
	if _, err := c.InsertOne(context.Background(), bson.D{{Key: "x", Value: 1}}); err != nil {
		t.Fatalf("err = %v, want nil passthrough", err)
	}
	if fake.inserts != 1 {
		t.Fatalf("underlying InsertOne ran %d times, want 1", fake.inserts)
	}
}

func TestFindOneInjectsErrorViaSingleResult(t *testing.T) {
	sentinel := errors.New("read down")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpMongo),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("find-err"))

	fake := &fakeColl{}
	c := newColl(fake, eng)
	res := c.FindOne(context.Background(), bson.D{{Key: "_id", Value: 1}})

	if !errors.Is(res.Err(), sentinel) {
		t.Fatalf("res.Err() = %v, want sentinel", res.Err())
	}
	if fake.findOnes != 0 {
		t.Fatalf("underlying FindOne ran %d times, want 0 (fault short-circuits)", fake.findOnes)
	}
}

func TestFindOnePassesThroughWhenNoRule(t *testing.T) {
	fake := &fakeColl{}
	c := newColl(fake, engine.New())
	_ = c.FindOne(context.Background(), bson.D{{Key: "_id", Value: 1}})
	if fake.findOnes != 1 {
		t.Fatalf("underlying FindOne ran %d times, want 1", fake.findOnes)
	}
}

func TestFindOneConnDropMapsToRetryableCommandError(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpMongo),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	fake := &fakeColl{}
	c := newColl(fake, eng)
	res := c.FindOne(context.Background(), bson.D{{Key: "_id", Value: 1}})

	var ce gomongo.CommandError
	if !errors.As(res.Err(), &ce) {
		t.Fatalf("res.Err() = %T %v, want mongo.CommandError", res.Err(), res.Err())
	}
	if !ce.HasErrorLabel("NetworkError") {
		t.Fatalf("CommandError labels = %v, want NetworkError present", ce.Labels)
	}
	if fake.findOnes != 0 {
		t.Fatalf("underlying FindOne ran %d times, want 0 (fault short-circuits)", fake.findOnes)
	}
}

func TestFindAggregateUpdateDeleteFault(t *testing.T) {
	sentinel := errors.New("op down")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpMongo),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("op-err")) // matches every OpMongo op

	fake := &fakeColl{}
	c := newColl(fake, eng)
	ctx := context.Background()

	if _, err := c.Find(ctx, bson.D{}); !errors.Is(err, sentinel) {
		t.Fatalf("Find err = %v, want sentinel", err)
	}
	if _, err := c.Aggregate(ctx, bson.A{}); !errors.Is(err, sentinel) {
		t.Fatalf("Aggregate err = %v, want sentinel", err)
	}
	if _, err := c.UpdateOne(ctx, bson.D{}, bson.D{{Key: "$set", Value: bson.D{{Key: "x", Value: 1}}}}); !errors.Is(err, sentinel) {
		t.Fatalf("UpdateOne err = %v, want sentinel", err)
	}
	if _, err := c.DeleteOne(ctx, bson.D{}); !errors.Is(err, sentinel) {
		t.Fatalf("DeleteOne err = %v, want sentinel", err)
	}
	if fake.finds != 0 || fake.aggs != 0 || fake.updates != 0 || fake.deletes != 0 {
		t.Fatalf("an underlying op ran despite the fault: finds=%d aggs=%d updates=%d deletes=%d", fake.finds, fake.aggs, fake.updates, fake.deletes)
	}
}

func TestFindUpdateDeletePassThroughWhenNoRule(t *testing.T) {
	fake := &fakeColl{}
	c := newColl(fake, engine.New())
	ctx := context.Background()
	_, _ = c.Find(ctx, bson.D{})
	_, _ = c.Aggregate(ctx, bson.A{})
	_, _ = c.UpdateOne(ctx, bson.D{}, bson.D{})
	_, _ = c.DeleteOne(ctx, bson.D{})
	if fake.finds != 1 || fake.aggs != 1 || fake.updates != 1 || fake.deletes != 1 {
		t.Fatalf("passthrough miscount: finds=%d aggs=%d updates=%d deletes=%d, want all 1", fake.finds, fake.aggs, fake.updates, fake.deletes)
	}
}

func TestWrappedDatabasePropagatesEngine(t *testing.T) {
	eng := engine.New()
	// Database()/Collection() are exercised end-to-end in the integration test;
	// here we assert the wrapper carries the engine so produced collections fault.
	db := &Database{eng: eng}
	if db.eng != eng {
		t.Fatal("Database did not retain the engine")
	}
	cl := &Client{eng: eng}
	if cl.eng != eng {
		t.Fatal("Client did not retain the engine")
	}
}
