// Package kafka is a chaos adapter for github.com/segmentio/kafka-go. The
// kafka-go Reader and Writer are concrete structs with no hook interface, so this
// adapter wraps them into its own types that consult the chaotic engine on each
// read, fetch, commit, and write:
//
//	import chaoskafka "github.com/ag4r/chaotic/adapter/kafka"
//
//	r := kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, Topic: "events", GroupID: "g"})
//	cr := chaoskafka.WrapReader(r, eng)
//	msg, err := cr.ReadMessage(ctx)
//
// *Reader embeds *kafka.Reader and *Writer embeds *kafka.Writer, so every method
// this adapter does not fault (Stats, Lag, Offset, Close, SetOffset, ...) passes
// through unchanged and the wrappers are drop-in for the kafka-go types.
//
// Faulted methods (v1): Reader.ReadMessage, Reader.FetchMessage,
// Reader.CommitMessages, Writer.WriteMessages. WriteMessages faults once per call,
// not per message.
//
// Fault mapping (faults stay in kafka-go's native model):
//
//	fault.Latency / Jittered  -> ctx-honoring sleep, then the real op runs
//	fault.Error(err)          -> err is returned as-is
//	fault.ConnDrop()          -> io.ErrUnexpectedEOF, which kafka-go treats as a transport error and retries
//	fault.Panic(v)            -> panic(v)
//
// Build with -tags chaos_off to compile the wrapper out entirely: the faulted
// methods become zero-allocation passthroughs to the wrapped reader/writer.
package kafka
