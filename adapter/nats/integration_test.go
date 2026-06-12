//go:build !chaos_off

package nats_test

import (
	"errors"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/test"
	natsgo "github.com/nats-io/nats.go"

	chaosnats "github.com/RomanAgaltsev/chaotic/adapter/nats"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestWrapperAgainstInProcessServer(t *testing.T) {
	srv := natsserver.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)

	nc, err := natsgo.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(nc.Close)

	eng := engine.New()
	cc := chaosnats.WrapConn(nc, eng)

	// Happy path: subscribe (via the raw conn so it is not faulted) and publish
	// through the wrapper.
	got := make(chan string, 1)
	if _, err := nc.Subscribe("events", func(m *natsgo.Msg) { got <- string(m.Data) }); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if err := cc.Publish("events", []byte("hello")); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := cc.Flush(); err != nil { // promoted, un-faulted method
		t.Fatalf("Flush: %v", err)
	}
	select {
	case msg := <-got:
		if msg != "hello" {
			t.Fatalf("received %q, want \"hello\"", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive the published message")
	}

	// Fault path: a rule that fails the publish surfaces the supplied error.
	sentinel := errors.New("chaos")
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNATS),
		engine.MatchName("events"),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("pub-fail"))

	if err := cc.Publish("events", []byte("hello")); !errors.Is(err, sentinel) {
		t.Fatalf("Publish err = %v, want sentinel", err)
	}
}
