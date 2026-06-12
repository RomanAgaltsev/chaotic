//go:build integration && !chaos_off

package mongo_test

import (
	"context"
	"errors"
	"testing"

	tcmongo "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	chaosmongo "github.com/RomanAgaltsev/chaotic/adapter/mongo"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestWrapperAgainstRealMongo(t *testing.T) {
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
	cc := chaosmongo.WrapClient(client, eng)
	coll := cc.Database("app").Collection("users") // auto-wrapped *Collection

	// Happy path: insert, then read it back via FindOne.
	if _, err := coll.InsertOne(ctx, bson.D{{Key: "_id", Value: 1}, {Key: "name", Value: "ada"}}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}
	var got struct {
		Name string `bson:"name"`
	}
	if err := coll.FindOne(ctx, bson.D{{Key: "_id", Value: 1}}).Decode(&got); err != nil || got.Name != "ada" {
		t.Fatalf("FindOne = %q err=%v; want \"ada\"", got.Name, err)
	}

	// Promoted (un-faulted) method passes through: CountDocuments.
	if n, err := coll.CountDocuments(ctx, bson.D{}); err != nil || n != 1 {
		t.Fatalf("CountDocuments = %d err=%v; want 1", n, err)
	}

	// Fault path: a rule that fails the insert surfaces as the supplied CommandError.
	sentinel := mongo.CommandError{Code: 91, Message: "chaos", Labels: []string{"NetworkError"}}
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpMongo),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("op-fail"))

	_, err = coll.InsertOne(ctx, bson.D{{Key: "_id", Value: 2}})
	var ce mongo.CommandError
	if !errors.As(err, &ce) {
		t.Fatalf("InsertOne err = %T %v, want mongo.CommandError", err, err)
	}
}
