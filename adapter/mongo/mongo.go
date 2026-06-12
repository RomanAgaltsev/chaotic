//go:build !chaos_off

package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	gomongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// Collection wraps a *mongo.Collection so the chaotic engine is consulted on each
// faultable query/write. Every other *mongo.Collection method is promoted from the
// embedded value, so a *Collection is a drop-in for *mongo.Collection.
type Collection struct {
	*gomongo.Collection
	coll mongoColl
	eng  *engine.Engine
	db   string
	name string
}

// WrapCollection returns a *Collection that consults eng on each faultable method.
func WrapCollection(coll *gomongo.Collection, eng *engine.Engine) *Collection {
	return &Collection{
		Collection: coll,
		coll:       coll,
		eng:        eng,
		db:         coll.Database().Name(),
		name:       coll.Name(),
	}
}

// op builds the Op for this collection with the given command name.
func (c *Collection) op(name string) engine.Op {
	return engine.Op{
		Kind:   engine.OpMongo,
		Name:   name,
		Method: "command",
		Attrs:  map[string]string{"db": c.db, "coll": c.name},
	}
}

// before runs the pre-call lifecycle. A non-nil returned error is the mapped
// fault: the caller must return it (or wrap it into the method's result type)
// WITHOUT making the real call, and must not call reportOutcome again — before
// already reported the injected fault. Otherwise the caller makes the real call
// and then calls reportOutcome(ctx, action, callErr) exactly once.
func (c *Collection) before(ctx context.Context, op engine.Op) (engine.Action, error) {
	action := c.eng.Eval(ctx, op)
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return nil, mapErr(err)
	}
	return action, nil
}

// InsertOne faults the insert path, then delegates to the real collection.
func (c *Collection) InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*gomongo.InsertOneResult, error) {
	if !c.eng.Enabled() {
		return c.coll.InsertOne(ctx, document, opts...)
	}
	action, ferr := c.before(ctx, c.op("insert"))
	if ferr != nil {
		return nil, ferr
	}
	res, err := c.coll.InsertOne(ctx, document, opts...)
	reportOutcome(ctx, action, err)
	return res, err
}

// FindOne faults the single-document read. The fault is delivered through the
// returned SingleResult's Err()/Decode(), matching how the driver reports a
// failed FindOne.
func (c *Collection) FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *gomongo.SingleResult {
	if !c.eng.Enabled() {
		return c.coll.FindOne(ctx, filter, opts...)
	}
	action, ferr := c.before(ctx, c.op("find"))
	if ferr != nil {
		return errSingleResult(ferr)
	}
	res := c.coll.FindOne(ctx, filter, opts...)
	reportOutcome(ctx, action, res.Err())
	return res
}

// Find faults the multi-document read, then delegates.
func (c *Collection) Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*gomongo.Cursor, error) {
	if !c.eng.Enabled() {
		return c.coll.Find(ctx, filter, opts...)
	}
	action, ferr := c.before(ctx, c.op("find"))
	if ferr != nil {
		return nil, ferr
	}
	cur, err := c.coll.Find(ctx, filter, opts...)
	reportOutcome(ctx, action, err)
	return cur, err
}

// Aggregate faults the aggregation pipeline at open. Per-stage faults are deferred
// to the per-row primitive (see design §4.1).
func (c *Collection) Aggregate(ctx context.Context, pipeline any, opts ...options.Lister[options.AggregateOptions]) (*gomongo.Cursor, error) {
	if !c.eng.Enabled() {
		return c.coll.Aggregate(ctx, pipeline, opts...)
	}
	action, ferr := c.before(ctx, c.op("aggregate"))
	if ferr != nil {
		return nil, ferr
	}
	cur, err := c.coll.Aggregate(ctx, pipeline, opts...)
	reportOutcome(ctx, action, err)
	return cur, err
}

// UpdateOne faults the single-document update, then delegates.
func (c *Collection) UpdateOne(ctx context.Context, filter, update any, opts ...options.Lister[options.UpdateOneOptions]) (*gomongo.UpdateResult, error) {
	if !c.eng.Enabled() {
		return c.coll.UpdateOne(ctx, filter, update, opts...)
	}
	action, ferr := c.before(ctx, c.op("update"))
	if ferr != nil {
		return nil, ferr
	}
	res, err := c.coll.UpdateOne(ctx, filter, update, opts...)
	reportOutcome(ctx, action, err)
	return res, err
}

// DeleteOne faults the single-document delete, then delegates.
func (c *Collection) DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*gomongo.DeleteResult, error) {
	if !c.eng.Enabled() {
		return c.coll.DeleteOne(ctx, filter, opts...)
	}
	action, ferr := c.before(ctx, c.op("delete"))
	if ferr != nil {
		return nil, ferr
	}
	res, err := c.coll.DeleteOne(ctx, filter, opts...)
	reportOutcome(ctx, action, err)
	return res, err
}

// mapErr translates a fault error into the driver's native error model. ConnDrop
// becomes a mongo.CommandError carrying the NetworkError/RetryableWriteError
// labels so the driver's retryable-write/read logic engages; every other fault
// error passes through unchanged, so a caller who wants a specific command error
// supplies fault.Error(mongo.CommandError{...}).
func mapErr(err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return gomongo.CommandError{
			Code:    6, // HostUnreachable
			Message: "chaotic: connection drop",
			Labels:  []string{"NetworkError", "RetryableWriteError"},
		}
	}
	return err
}

// reportOutcome forwards the call's error (or the injected fault) to the engine
// when the action reports outcomes, then runs After to release any held bound
// (e.g. a WithMaxConcurrent slot). Call it exactly once per action, or the slot
// leaks and the failure budget never sees the call. A nil action is a no-op.
func reportOutcome(ctx context.Context, action engine.Action, callErr error) {
	if action == nil {
		return
	}
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
	_ = action.After(ctx)
}

// errSingleResult builds a *mongo.SingleResult carrying err, for faulting FindOne
// (whose signature returns no error — the error lives inside the SingleResult).
func errSingleResult(err error) *gomongo.SingleResult {
	return gomongo.NewSingleResultFromDocument(bson.D{}, err, nil)
}

// Database wraps a *mongo.Database so the collections it hands out are already
// chaos-wrapped. Every other *mongo.Database method is promoted from the embedded
// value.
type Database struct {
	*gomongo.Database
	eng *engine.Engine
}

// WrapDatabase returns a *Database whose Collection returns chaos-wrapped collections.
func WrapDatabase(db *gomongo.Database, eng *engine.Engine) *Database {
	return &Database{Database: db, eng: eng}
}

// Collection returns an already chaos-wrapped *Collection.
func (d *Database) Collection(name string, opts ...options.Lister[options.CollectionOptions]) *Collection {
	return WrapCollection(d.Database.Collection(name, opts...), d.eng)
}

// Client wraps a *mongo.Client so the databases (and thus collections) it hands
// out are already chaos-wrapped. Every other *mongo.Client method is promoted.
type Client struct {
	*gomongo.Client
	eng *engine.Engine
}

// WrapClient returns a *Client whose Database returns chaos-wrapped databases.
func WrapClient(client *gomongo.Client, eng *engine.Engine) *Client {
	return &Client{Client: client, eng: eng}
}

// Database returns an already chaos-wrapped *Database.
func (c *Client) Database(name string, opts ...options.Lister[options.DatabaseOptions]) *Database {
	return WrapDatabase(c.Client.Database(name, opts...), c.eng)
}
