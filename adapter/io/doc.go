// Package io brings chaos to io.Reader / io.Writer boundaries: file I/O, request
// and response body streaming, any pipe. Wrap a stream with WrapReader/WrapWriter
// and the engine is consulted on each Read/Write.
//
// Stream-shaping faults shape the data instead of aborting:
//
//	fault.SlowReader(rate) / fault.SlowWriter(rate) - trickle bytes at rate B/s
//	                                                  (rate 0 blocks: a stream that never ends)
//	fault.Truncate(n)                               - cut the stream after n bytes
//	                                                  (reader -> io.EOF, writer -> io.ErrShortWrite)
//
// Other faults map as usual: fault.Error is returned as the read/write error.
// Because io.Reader/io.Writer carry no context, slow/block sleeps use a background
// context and are not cancellable from the call (as in adapter/net).
//
// Imported as chaosio.
package io
