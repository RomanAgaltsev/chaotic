// Package mongo is a chaos adapter for go.mongodb.org/mongo-driver/v2. The
// driver's CommandMonitor and PoolMonitor are observation-only and cannot fail a
// command, so this adapter wraps the collection (and, for auto-wrap, the database
// and client) into its own types that consult the chaotic engine on each query
// and write:
//
//	import chaosmongo "github.com/RomanAgaltsev/chaotic/adapter/mongo"
//
//	coll := client.Database("app").Collection("users")
//	cc := chaosmongo.WrapCollection(coll, eng)
//	res := cc.FindOne(ctx, bson.D{{"_id", id}})
//
// *Collection embeds *mongo.Collection, so every method this adapter does not
// fault (CountDocuments, InsertMany, Indexes, Name, ...) passes through unchanged
// and a *Collection is a drop-in for *mongo.Collection.
//
// Faulted methods (v1): FindOne, Find, InsertOne, UpdateOne, DeleteOne, Aggregate.
// FindOne and Find both report as the "find" command, so a MatchName("find")
// rule targets both single- and multi-document reads.
//
// Fault mapping (faults stay in the driver's native error model):
//
//	fault.Latency / Jittered  -> ctx-honoring sleep, then the real op runs
//	fault.Error(err)          -> err is returned as-is (supply mongo.CommandError{...} for native handling)
//	fault.ConnDrop()          -> mongo.CommandError labeled NetworkError/RetryableWriteError, so retryable-write logic engages
//	fault.Panic(v)            -> panic(v)
//
// Build with -tags chaos_off to compile the wrapper out entirely: the faulted
// methods become zero-allocation passthroughs to the wrapped collection.
package mongo
