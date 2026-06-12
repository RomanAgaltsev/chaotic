package main

import (
	"encoding/json"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestTruncatedBodyIsHandled(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpIO),
		engine.Times(1),
		engine.WithFault(fault.Truncate(4)),
	).Named("trunc"))

	got, err := ReadBody(eng, `{"ok":true}`)
	if err != nil {
		t.Fatalf("ReadBody err = %v", err)
	}
	// The body was cut to 4 bytes: JSON parsing must fail cleanly, not panic.
	var v map[string]any
	if json.Unmarshal(got, &v) == nil {
		t.Fatalf("expected truncated JSON %q to fail parsing", got)
	}
}
