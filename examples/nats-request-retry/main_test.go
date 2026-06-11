package main

import (
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/test"
	natsgo "github.com/nats-io/nats.go"

	chaosnats "github.com/ag4r/chaotic/adapter/nats"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestRequestRetrySurvivesOutage(t *testing.T) {
	srv := natsserver.RunRandClientPortServer()
	t.Cleanup(srv.Shutdown)

	nc, err := natsgo.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(nc.Close)

	// Responder on the raw connection (not faulted): echoes the request.
	if _, err := nc.Subscribe("svc.echo", func(m *natsgo.Msg) { _ = m.Respond([]byte("pong")) }); err != nil {
		t.Fatalf("responder Subscribe: %v", err)
	}

	eng := engine.New()
	cc := chaosnats.WrapConn(nc, eng)

	// Drop the first two requests: a transient outage. ConnDrop returns
	// nats.ErrConnectionClosed but never touches the real connection, so the
	// retry's next request lands on the still-open connection.
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNATS),
		engine.MatchName("svc.echo"),
		engine.Times(2),
		engine.WithFault(fault.ConnDrop()),
	).Named("outage"))

	msg, err := RequestWithRetry(cc, "svc.echo", []byte("ping"), 2*time.Second, 5)
	if err != nil {
		t.Fatalf("RequestWithRetry failed despite retries: %v", err)
	}
	if string(msg.Data) != "pong" {
		t.Fatalf("reply = %q, want \"pong\"", msg.Data)
	}
	if got := eng.Hits("outage"); got != 2 {
		t.Fatalf("outage fired %d times, want 2", got)
	}
}
