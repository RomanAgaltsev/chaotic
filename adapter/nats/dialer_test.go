//go:build !chaos_off

package nats

import (
	"errors"
	"testing"

	natsgo "github.com/nats-io/nats.go"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestChaosDialerFaultsBeforeNetwork(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNATS),
		engine.MatchName("localhost:4222"),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("dial-drop"))

	d := &chaosDialer{eng: eng}
	conn, err := d.Dial("tcp", "localhost:4222")

	if conn != nil {
		t.Fatalf("conn = %v, want nil", conn)
	}
	if !errors.Is(err, natsgo.ErrConnectionClosed) {
		t.Fatalf("err = %v, want nats.ErrConnectionClosed", err)
	}
}

func TestOptionReturnsUsableOption(t *testing.T) {
	// Option(eng) must produce a nats.Option that applies cleanly to nats.Options.
	var opts natsgo.Options
	if err := Option(engine.New())(&opts); err != nil {
		t.Fatalf("applying Option: %v", err)
	}
	if opts.CustomDialer == nil {
		t.Fatal("Option did not install a CustomDialer")
	}
}
