package main

import (
	"context"
	"testing"

	tcmongo "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	chaosmongo "github.com/RomanAgaltsev/chaotic/adapter/mongo"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestReadRetrySurvivesFailover(t *testing.T) {
	ctx := context.Background()

	ctr, err := tcmongo.Run(ctx, "mongo:7")
	if err != nil {
		t.Skipf("cannot start MongoDB container (Docker unavailable?): %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	uri, err := ctr.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("ConnectionString: %v", err)
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(ctx) })

	eng := engine.New()
	coll := chaosmongo.WrapClient(client, eng).Database("app").Collection("users")

	if _, err := coll.InsertOne(ctx, bson.D{{Key: "_id", Value: 1}, {Key: "name", Value: "ada"}}); err != nil {
		t.Fatalf("seed InsertOne: %v", err)
	}

	// Drop the first two reads: a transient failover. ConnDrop returns a labeled
	// CommandError but never touches the real collection, so the retry's next read
	// lands on the still-connected client.
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpMongo),
		engine.Times(2),
		engine.WithFault(fault.ConnDrop()),
	).Named("failover"))

	u, err := ReadUserWithRetry(ctx, coll, 1, 5)
	if err != nil {
		t.Fatalf("ReadUserWithRetry failed despite retries: %v", err)
	}
	if u.Name != "ada" {
		t.Fatalf("read user = %q, want \"ada\"", u.Name)
	}
	if got := eng.Hits("failover"); got != 2 {
		t.Fatalf("failover fired %d times, want 2", got)
	}
}
