//go:build chaos_off

// Package mongo (chaos_off build): the wrapper adds no behavior and no
// allocation. Faulted methods forward straight to the wrapped collection; every
// other method is promoted from the embedded *mongo.Collection / *mongo.Database
// / *mongo.Client.
package mongo

import (
	"context"

	gomongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/RomanAgaltsev/chaotic/engine"
)

type Collection struct {
	*gomongo.Collection
	coll mongoColl
}

func WrapCollection(coll *gomongo.Collection, _ *engine.Engine) *Collection {
	return &Collection{Collection: coll, coll: coll}
}

func (c *Collection) FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *gomongo.SingleResult {
	return c.coll.FindOne(ctx, filter, opts...)
}

func (c *Collection) Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*gomongo.Cursor, error) {
	return c.coll.Find(ctx, filter, opts...)
}

func (c *Collection) InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*gomongo.InsertOneResult, error) {
	return c.coll.InsertOne(ctx, document, opts...)
}

func (c *Collection) UpdateOne(ctx context.Context, filter, update any, opts ...options.Lister[options.UpdateOneOptions]) (*gomongo.UpdateResult, error) {
	return c.coll.UpdateOne(ctx, filter, update, opts...)
}

func (c *Collection) DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*gomongo.DeleteResult, error) {
	return c.coll.DeleteOne(ctx, filter, opts...)
}

func (c *Collection) Aggregate(ctx context.Context, pipeline any, opts ...options.Lister[options.AggregateOptions]) (*gomongo.Cursor, error) {
	return c.coll.Aggregate(ctx, pipeline, opts...)
}

type Database struct {
	*gomongo.Database
}

func WrapDatabase(db *gomongo.Database, _ *engine.Engine) *Database {
	return &Database{Database: db}
}

func (d *Database) Collection(name string, opts ...options.Lister[options.CollectionOptions]) *Collection {
	return WrapCollection(d.Database.Collection(name, opts...), nil)
}

type Client struct {
	*gomongo.Client
}

func WrapClient(client *gomongo.Client, _ *engine.Engine) *Client {
	return &Client{Client: client}
}

func (c *Client) Database(name string, opts ...options.Lister[options.DatabaseOptions]) *Database {
	return WrapDatabase(c.Client.Database(name, opts...), nil)
}
