// Command mongo-read-fallback demonstrates a read that retries through a transient
// MongoDB failover (primary step-down), and proves the retry works using the
// chaotic adapter/mongo wrapper to fail reads on demand — no real outage, no flaky
// timing.
package main

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	chaosmongo "github.com/ag4r/chaotic/adapter/mongo"
)

// User is the document we read back.
type User struct {
	ID   int    `bson:"_id"`
	Name string `bson:"name"`
}

// ReadUserWithRetry reads a user by id, retrying up to attempts times with a short
// linear backoff so a transient failover does not surface as a hard error.
func ReadUserWithRetry(ctx context.Context, coll *chaosmongo.Collection, id, attempts int) (User, error) {
	var u User
	var err error
	for i := range attempts {
		err = coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&u)
		if err == nil {
			return u, nil
		}
		time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
	}
	return u, err
}

func main() {
	fmt.Println("run `go test` in this directory to see read-with-retry survive a chaos failover")
}
