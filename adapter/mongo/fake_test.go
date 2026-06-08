package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Preallocated results shared across fake calls. The fake never mutates them and
// callers only inspect the returned error, so sharing a pointer keeps the fake
// allocation-free (required by the chaos_off zero-alloc gate).
var (
	fakeInsertOneResult = &mongo.InsertOneResult{}
	fakeUpdateResult    = &mongo.UpdateResult{}
	fakeDeleteResult    = &mongo.DeleteResult{}
)

// fakeColl is a zero-network mongoColl for unit tests: it records call counts and
// returns preconfigured errors. A zero fakeColl succeeds on every method.
type fakeColl struct {
	findErr   error
	insertErr error
	updateErr error
	deleteErr error
	aggErr    error

	finds    int
	inserts  int
	updates  int
	deletes  int
	aggs     int
	findOnes int
}

func (f *fakeColl) FindOne(_ context.Context, _ any, _ ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
	f.findOnes++
	return mongo.NewSingleResultFromDocument(bson.D{}, f.findErr, nil)
}

func (f *fakeColl) Find(_ context.Context, _ any, _ ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	f.finds++
	return nil, f.findErr
}

func (f *fakeColl) InsertOne(context.Context, any, ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	f.inserts++
	return fakeInsertOneResult, f.insertErr
}

func (f *fakeColl) UpdateOne(_ context.Context, _ any, _ any, _ ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	f.updates++
	return fakeUpdateResult, f.updateErr
}

func (f *fakeColl) DeleteOne(_ context.Context, _ any, _ ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
	f.deletes++
	return fakeDeleteResult, f.deleteErr
}

func (f *fakeColl) Aggregate(context.Context, any, ...options.Lister[options.AggregateOptions]) (*mongo.Cursor, error) {
	f.aggs++
	return nil, f.aggErr
}
