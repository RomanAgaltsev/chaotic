// Package net is a chaos adapter for the standard library net package. It wraps
// net.Conn, net.Listener, and the dial step so the chaotic engine is consulted at
// the raw byte boundary — chaos for anything that speaks TCP, without a
// per-library adapter:
//
//	import chaosnet "github.com/RomanAgaltsev/chaotic/adapter/net"
//
//	conn = chaosnet.WrapConn(conn, eng)             // fault Read/Write
//	ln   = chaosnet.WrapListener(ln, eng)           // accepted conns auto-wrapped
//	d    := chaosnet.WrapDialer(eng, nil)           // fault at dial; dialed conn wrapped
//	c, _ := d.DialContext(ctx, "tcp", addr)
//
// The wrappers embed the net interface value, so every method this adapter does
// not fault (Close, LocalAddr, SetDeadline, ...) passes through unchanged.
//
// Fault mapping (faults stay in net's model):
//
//	fault.Latency / Jittered  -> sleep, then the real op (Read/Write have no ctx)
//	fault.Error(err)          -> returned from Read/Write/DialContext
//	fault.ConnDrop()          -> *net.OpError wrapping io.ErrUnexpectedEOF (never auto-closes)
//	fault.Panic(v)            -> panic(v)
//	fault.Disconnect()		  -> *net.OpError wrapping io.EOF
//
// Byte-rate faults (SlowReader/SlowWriter, per-byte) pair with this adapter but
// are a later addition. Build with -tags chaos_off to compile the wrappers out:
// WrapConn returns the conn unchanged and adds zero allocations.
package net
