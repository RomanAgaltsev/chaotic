package nats

import (
	"time"

	natsgo "github.com/nats-io/nats.go"
)

// fakeConn is a zero-network natsConn for unit tests: it records call counts and
// returns preconfigured errors. A zero fakeConn succeeds on every method.
type fakeConn struct {
	publishErr   error
	requestErr   error
	subscribeErr error
	drainErr     error

	publishes  int
	requests   int
	subscribes int
	queues     int
	drains     int
}

func (f *fakeConn) Publish(string, []byte) error { f.publishes++; return f.publishErr }

func (f *fakeConn) Request(string, []byte, time.Duration) (*natsgo.Msg, error) {
	f.requests++
	if f.requestErr != nil {
		return nil, f.requestErr
	}
	return &natsgo.Msg{}, nil
}

func (f *fakeConn) Subscribe(string, natsgo.MsgHandler) (*natsgo.Subscription, error) {
	f.subscribes++
	return nil, f.subscribeErr
}

func (f *fakeConn) QueueSubscribe(string, string, natsgo.MsgHandler) (*natsgo.Subscription, error) {
	f.queues++
	return nil, f.subscribeErr
}

func (f *fakeConn) Drain() error { f.drains++; return f.drainErr }
