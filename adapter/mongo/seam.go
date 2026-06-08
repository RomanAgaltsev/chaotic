package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// mongoColl is the faultable subset of *mongo.Collection that the chaos wrapper
// intercepts. *mongo.Collection satisfies it directly; unit tests supply a fake.
// Every other *mongo.Collection method (CountDocuments, InsertMany, Indexes,
// Name, ...) reaches callers unchanged via the embedded *mongo.Collection.
type mongoColl interface {
	FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult
	Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error)
	InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
	UpdateOne(ctx context.Context, filter, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
	DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error)
	Aggregate(ctx context.Context, pipeline any, opts ...options.Lister[options.AggregateOptions]) (*mongo.Cursor, error)
}
