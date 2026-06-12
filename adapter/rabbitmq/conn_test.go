//go:build !chaos_off

package rabbitmq

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestConnChannelConnDropWithoutTouchingConn(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.MatchName("channel"),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("chan-drop"))

	// nil connection: the fault must fire before the connection is dereferenced.
	cc := WrapConnection(nil, eng)
	ch, err := cc.Channel()

	if ch != nil {
		t.Fatalf("ch = %v, want nil", ch)
	}
	if !errors.Is(err, amqp.ErrClosed) {
		t.Fatalf("err = %v, want amqp.ErrClosed", err)
	}
}

func TestConnChannelErrorWithoutTouchingConn(t *testing.T) {
	sentinel := errors.New("no channels")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("chan-err"))

	cc := WrapConnection(nil, eng)
	if _, err := cc.Channel(); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
	_ = context.Background()
}
