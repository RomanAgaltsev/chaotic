// Command redis-cache-fallback demonstrates a read-through cache that falls
// back to a database when Redis is unavailable, and shows how to prove that
// resilience with the chaotic redis adapter.
package main

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// Store is a tiny read-through cache: it reads from Redis, and on any Redis
// error falls back to the backing database (and best-effort backfills the cache).
type Store struct {
	rc *goredis.Client
	db map[string]string // stand-in for a real database
}

func NewStore(rc *goredis.Client, db map[string]string) *Store {
	return &Store{rc: rc, db: db}
}

// Get returns the value for key, surviving a Redis outage by serving from db.
func (s *Store) Get(ctx context.Context, key string) (string, bool) {
	if v, err := s.rc.Get(ctx, key).Result(); err == nil {
		return v, true
	}
	// Redis errored (miss or outage): fall back to the database.
	v, ok := s.db[key]
	if ok {
		_ = s.rc.Set(ctx, key, v, 0).Err() // best-effort backfill; ignore chaos here
	}
	return v, ok
}

func main() {
	fmt.Println("run `go test` in this directory to see the fallback under chaos")
}
