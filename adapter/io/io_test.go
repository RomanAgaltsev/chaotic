//go:build !chaos_off

package io_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

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
