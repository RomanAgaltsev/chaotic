// Package redis is a chaos adapter for github.com/redis/go-redis/v9. Install
// the hook returned by NewHook on any go-redis client and the chaotic engine
// is consulted on every command, on every pipeline, and on dial:
//
//	import chaosredis "github.com/RomanAgaltsev/chaotic/adapter/redis"
//
//	rc := redis.NewClient(opts)
//	rc.AddHook(chaosredis.NewHook(eng))
//
// The hook is stateless with respect to client identity, so one hook value may
// be installed on any number of clients — *redis.Client, *redis.ClusterClient,
// and sentinel-backed clients alike.
//
// Fault mapping (faults stay in go-redis's native error model):
//
//	fault.Latency / Jittered  -> ctx-honoring sleep, then the real command runs
//	fault.Error(err)          -> err is set on the command (cmd.Err())
//	fault.ConnDrop()          -> io.ErrUnexpectedEOF wrapped in *net.OpError
//	fault.Panic(v)            -> panic(v)
//
// Build the binary with -tags chaos_off to compile the hook out entirely: NewHook
// then returns a passthrough that adds zero allocations to the command path.
package redis
