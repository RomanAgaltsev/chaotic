package scenarios

import (
	"time"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// Option tunes a scenario. The same three options apply to every scenario;
// each scenario uses the ones relevant to it.
type Option func(*config)

type config struct {
	latency   time.Duration
	errorRate float64
	count     int
}

func defaults() config {
	return config{
		latency:   200 * time.Millisecond,
		errorRate: 0.5,
		count:     5,
	}
}

func apply(opts []Option) config {
	c := defaults()
	for _, o := range opts {
		o(&c)
	}
	return c
}

// WithLatency sets the injected latency for scenarios that slow calls.
func WithLatency(d time.Duration) Option { return func(c *config) { c.latency = d } }

// WithErrorRate sets the probability [0,1] for scenarios that fail a fraction
// of calls.
func WithErrorRate(p float64) Option { return func(c *config) { c.errorRate = p } }

// WithCount sets the number of affected calls for scenarios that fault a fixed
// burst.
func WithCount(n int) Option { return func(c *config) { c.count = n } }

// DatabaseOutageCascade drops the first WithCount SQL/pgx calls (the outage),
// then injects WithLatency on the next WithCount calls (the recovery lag as the
// pool re-warms). Models a database that goes away and comes back slow.
func DatabaseOutageCascade(eng *engine.Engine, opts ...Option) {
	c := apply(opts)
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpSQL, engine.OpPGX),
		engine.Times(c.count),
		engine.WithFault(fault.ConnDrop()),
	).Named("scenarios/db-outage"))
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpSQL, engine.OpPGX),
		engine.Range(c.count+1, 2*c.count),
		engine.WithFault(fault.Latency(c.latency)),
	).Named("scenarios/db-recovery-lag"))
}

// ThunderingHerdAfterDeploy returns HTTP 503 from a WithErrorRate fraction of
// inbound server requests, modeling a freshly deployed, cold service shedding
// load under a traffic spike.
func ThunderingHerdAfterDeploy(eng *engine.Engine, opts ...Option) {
	c := apply(opts)
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.Probability(c.errorRate, 0),
		engine.WithFault(fault.HTTPStatus(503)),
	).Named("scenarios/herd-503"))
}

// SlowLeaderElection injects WithLatency on the first WithCount Redis calls,
// modeling a coordination/lock backend that responds slowly during a contested
// leader election.
func SlowLeaderElection(eng *engine.Engine, opts ...Option) {
	c := apply(opts)
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.Times(c.count),
		engine.WithFault(fault.Latency(c.latency)),
	).Named("scenarios/slow-election"))
}

// PartialNetworkPartition drops a WithErrorRate fraction of outbound gRPC and
// HTTP client calls, modeling a partition where some peers are unreachable while
// others are fine.
func PartialNetworkPartition(eng *engine.Engine, opts ...Option) {
	c := apply(opts)
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient, engine.OpHTTPClient),
		engine.Probability(c.errorRate, 0),
		engine.WithFault(fault.ConnDrop()),
	).Named("scenarios/partition"))
}
