// Package rabbitmq is a chaos adapter for github.com/rabbitmq/amqp091-go. The
// amqp091-go client exposes no hook interface, so this adapter wraps the channel
// (and, for open-time faults, the connection) into its own types that consult the
// chaotic engine on each publish, consume, and ack/nack:
//
//	import chaosrabbitmq "github.com/ag4r/chaotic/adapter/rabbitmq"
//
//	conn, _ := amqp.Dial(url)
//	cc := chaosrabbitmq.WrapConnection(conn, eng)
//	ch, _ := cc.Channel()                       // already chaos-wrapped
//	_ = ch.PublishWithContext(ctx, ex, key, false, false, msg)
//
// *Channel embeds *amqp.Channel, so every method this adapter does not fault
// (QueueDeclare, ExchangeDeclare, Qos, Get, ...) passes through unchanged and a
// *Channel is a drop-in for *amqp.Channel.
//
// Fault mapping (faults stay in amqp091-go's native error model):
//
//	fault.Latency / Jittered  -> ctx-honoring sleep, then the real op runs
//	fault.Error(err)          -> err is returned as-is (supply &amqp.Error{...} for native handling)
//	fault.ConnDrop()          -> amqp.ErrClosed, so channel/connection recovery engages
//	fault.Panic(v)            -> panic(v)
//
// Per-delivery faults are out of scope for v1: a Delivery returned by Consume
// carries an Acknowledger bound to the underlying channel, so delivery.Ack()
// bypasses chaos. Use the channel-level Ack/Nack to exercise that path.
//
// Build with -tags chaos_off to compile the wrapper out entirely: the faultable
// methods become zero-allocation passthroughs to the wrapped channel.
package rabbitmq
