//go:build !chaos_off

package mongo

import (
	"context"
	"errors"
	"fmt"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ExampleWrapCollection() {
	// Fail the first insert, then recover.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpMongo),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("primary stepped down"))),
	).Named("mongo-flap"))

	// In production you wrap a live collection:
	//   cc := chaosmongo.WrapCollection(client.Database("app").Collection("users"), eng)
	// Here a fake stands in for the server so the example is hermetic.
	c := &Collection{coll: &fakeColl{}, eng: eng, db: "app", name: "users"}

	insert := func() error {
		_, err := c.InsertOne(context.Background(), bson.D{{Key: "name", Value: "ada"}})
		return err
	}

	fmt.Println("attempt 1:", insert())
	fmt.Println("attempt 2:", insert())
	// Output:
	// attempt 1: primary stepped down
	// attempt 2: <nil>
}
