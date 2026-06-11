package engine_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// fire evaluates op against eng and returns the injected fault error (or nil),
// mimicking what an adapter does around a wrapped call.
func fire(eng *engine.Engine, op engine.Op) error {
	ctx := context.Background()
	act := eng.Eval(ctx, op)
	err := act.Before(ctx)
	_ = act.After(ctx)
	return err
}

func ExampleNewRule() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.Times(1), // fire on the first match only
		engine.WithFault(fault.Error(errors.New("transient"))),
	).Named("flap"))

	op := engine.Op{
		Kind: engine.OpHTTPClient,
		Name: "/users",
	}
	fmt.Println("call 1:", fire(eng, op))
	fmt.Println("call 2:", fire(eng, op))
	fmt.Println("hits:", eng.Hits("flap"))
	// Output:
	// call 1: transient
	// call 2: <nil>
	// hits: 1
}

func ExampleMatchAttr() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.MatchAttr("host", "payments.internal"),
		engine.WithFault(fault.Error(errors.New("degraded"))),
	).Named("payments"))

	payments := engine.Op{
		Kind:  engine.OpHTTPClient,
		Attrs: map[string]string{"host": "payments.internal"},
	}

	search := engine.Op{
		Kind:  engine.OpHTTPClient,
		Attrs: map[string]string{"host": "search.internal"},
	}
	fmt.Println("payments:", fire(eng, payments))
	fmt.Println("search:", fire(eng, search))
	// Output:
	// payments: degraded
	// search: <nil>
}

func ExampleProbability() {
	// A seeded probability rule fires  identically every run, so chaos tests are
	// reproducible. Two engines built with the same seed produce same
	// fire/skip sequence.
	build := func() *engine.Engine {
		return engine.New().AddRule(engine.NewRule(
			engine.MatchKind(engine.OpHTTPClient),
			engine.Probability(0.5, 1),
			engine.WithFault(fault.Error(errors.New("x"))),
		).Named("p"))
	}
	seq := func(eng *engine.Engine) string {
		var b strings.Builder
		for range 10 {
			if fire(eng, engine.Op{Kind: engine.OpHTTPClient}) != nil {
				b.WriteByte('F')
			} else {
				b.WriteByte('.')
			}
		}
		return b.String()
	}
	first, second := seq(build()), seq(build())
	fmt.Println("reproducible:", first == second)
	// Output: reproducible: true
}

func ExampleBuildRule() {
	// BuildRule turns a declarative RuleSpec (e.g. decoded from YAML) into a
	// Rule. It is total: invalid specs return an error instead of panicking.
	rule, err := engine.BuildRule(engine.RuleSpec{
		Name:    "from-config",
		Kinds:   []string{"http_client"},
		Counter: engine.CounterSpec{Type: "times", N: 1},
		Faults:  []engine.FaultSpec{{Type: "error", Message: "boom"}},
	})
	if err != nil {
		fmt.Println("build error:", err)
		return
	}
	eng := engine.New().AddRule(rule)
	fmt.Println("fired:", fire(eng, engine.Op{Kind: engine.OpHTTPClient}) != nil)
	// Output: fired: true
}

func ExampleMatchTimeWindow() {
	// Only inject between 02:00 and 04:00 local time.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.MatchTimeWindow(2, 0, 4, 0),
		engine.WithFault(fault.Latency(0)),
	).Named("nightly"))
	fmt.Println(eng.Enabled())
	// Output: true
}

func ExampleMatchNameRegex() {
	// path.Match's * does not cross "/"; a regexp can.
	re := regexp.MustCompile(`^/api/v[0-9]+/users/.*$`)
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchNameRegex(re),
		engine.WithFault(fault.Error(errors.New("boom"))),
	).Named("users-api"))
	fmt.Println(eng.Enabled())
	// Output: true
}

func ExampleStickyAttr() {
	// Once a user trips the fault, keep them degraded for a minute.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.Probability(0.1, 1),
		engine.StickyAttr("user", time.Minute, 1024),
		engine.WithFault(fault.Error(errors.New("degraded"))),
	).Named("sticky-user"))
	fmt.Println(eng.Enabled())
	// Output: true
}

func ExampleWithPerRuleRateLimit() {
	// Fail Redis calls, but never more than 10 injected faults per second.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.WithPerRuleRateLimit(10),
		engine.WithFault(fault.ConnDrop()),
	).Named("redis-flaky"))
	fmt.Println(eng.Enabled())
	// Output: true
}
