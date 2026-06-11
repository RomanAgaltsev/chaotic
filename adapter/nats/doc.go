// Package nats is a chaos adapter for github.com/nats-io/nats.go. nats.go has no
// per-publish hook, so chaos arrives through two surfaces:
//
//	import chaosnats "github.com/ag4r/chaotic/adapter/nats"
//
//	// Construction-time: a chaos dialer faults Dial/reconnect (the ConnDrop story).
//	nc, _ := nats.Connect(url, chaosnats.Option(eng))
//
//	// Per-call: wrap the connection to fault Publish/Request/Subscribe.
//	cc := chaosnats.WrapConn(nc, eng)
//	err := cc.Publish("events", []byte("hi"))
//
// *Conn embeds *nats.Conn, so every method this adapter does not fault (Flush,
// Status, Close, LastError, ...) passes through unchanged and a *Conn is a drop-in
// for *nats.Conn.
//
// Faulted methods (v1): Conn.Publish, Conn.Request, Conn.Subscribe,
// Conn.QueueSubscribe, Conn.Drain, and the dialer's Dial. Subscription chaos
// faults at Subscribe open only; per-delivery faults are deferred. JetStream is a
// separate adapter.
//
// Fault mapping (faults stay in nats.go's native model):
//
//	fault.Latency / Jittered  -> sleep before the op
//	fault.Error(err)          -> err is returned as-is
//	fault.ConnDrop()          -> nats.ErrConnectionClosed (transient; the wrapper never calls nc.Close()),
//	                             so the caller's reconnect/retry path engages
//	fault.Panic(v)            -> panic(v)
//
// Build with -tags chaos_off to compile the wrapper out entirely: Option becomes a
// no-op nats.Option and the faulted methods become zero-allocation passthroughs.
package nats
