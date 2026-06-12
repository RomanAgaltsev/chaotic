//go:build !chaos_off

package io_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	chaosio "github.com/ag4r/chaotic/adapter/io"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestReadPassThroughWhenNoRule(t *testing.T) {
	eng := engine.New() // no rules
	r := chaosio.WrapReader(strings.NewReader("hello"), eng)
	got, err := io.ReadAll(r)
	if err != nil || string(got) != "hello" {
		t.Fatalf("ReadAll = (%q, %v), want (\"hello\", nil)", got, err)
	}
}

func TestReadReturnsErrorFault(t *testing.T) {
	sentinel := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpIO),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("err"))

	r := chaosio.WrapReader(strings.NewReader("hello"), eng)
	_, err := r.Read(make([]byte, 8))
	if !errors.Is(err, sentinel) {
		t.Fatalf("Read err = %v, want sentinel", err)
	}
}

func TestWritePassThroughWhenNoRule(t *testing.T) {
	eng := engine.New()
	var buf bytes.Buffer
	w := chaosio.WrapWriter(&buf, eng)
	n, err := w.Write([]byte("hello"))
	if err != nil || n != 5 || buf.String() != "hello" {
		t.Fatalf("Write = (%d, %v) buf=%q, want (5, nil) \"hello\"", n, err, buf.String())
	}
}

func TestSlowReaderDelaysButDeliversAll(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpIO),
		engine.Always(),
		engine.WithFault(fault.SlowReader(1000)), // 1000 B/s
	).Named("slow"))

	r := chaosio.WrapReader(strings.NewReader(strings.Repeat("x", 100)), eng)
	start := time.Now()
	got, err := io.ReadAll(r)
	if err != nil || len(got) != 100 {
		t.Fatalf("ReadAll = (%d bytes, %v), want (100, nil)", len(got), err)
	}
	if time.Since(start) < 50*time.Millisecond { // 100 B at 1000 B/s ~ 100ms total
		t.Fatalf("slow reader did not delay (elapsed %s)", time.Since(start))
	}
}

func TestSlowWriterMismatchOnReaderIsNoOp(t *testing.T) {
	// A SlowWriter sentinel firing on a reader shapes nothing but counts the hit.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpIO),
		engine.Always(),
		engine.WithFault(fault.SlowWriter(1)), // wrong direction for a reader
	).Named("mismatch"))

	r := chaosio.WrapReader(strings.NewReader("abc"), eng)
	start := time.Now()
	got, _ := io.ReadAll(r)
	if string(got) != "abc" {
		t.Fatalf("ReadAll = %q, want \"abc\"", got)
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatal("mismatched SlowWriter should not slow a reader")
	}
	if eng.Hits("mismatch") == 0 {
		t.Fatal("the rule should still record hits")
	}
}
