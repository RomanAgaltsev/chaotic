package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestBuildRuleConstructsMatchingRule(t *testing.T) {
	spec := RuleSpec{
		Name:     "transient",
		Kinds:    []string{"http_client"},
		NameGlob: "/users/*",
		Counter:  CounterSpec{Type: "times", N: 2},
		Faults:   []FaultSpec{{Type: "error", Message: "boom"}},
	}
	r, err := BuildRule(spec)
	if err != nil {
		t.Error(err)
	}
	if r.Name() != "transient" {
		t.Fatalf("name = %q, want transient", r.Name())
	}
	e := New().AddRule(r)
	if a := e.Eval(context.Background(), Op{Kind: OpHTTPClient, Name: "/users/123"}); a == Pass {
		t.Fatal("expected rule to fire for http_client /users/123")
	}
	if a := e.Eval(context.Background(), Op{Kind: OpSQL, Name: "/users/123"}); a != Pass {
		t.Fatal("expected no match for sql")
	}
}

func TestBuildRuleRejectsUnknownKind(t *testing.T) {
	if _, err := BuildRule(RuleSpec{Kinds: []string{"nope"}}); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestBuildRuleRejectsBadDuration(t *testing.T) {
	_, err := BuildRule(RuleSpec{Faults: []FaultSpec{{Type: "latency", Duration: "abc"}}})
	if err == nil {
		t.Fatal("expected error for unparseable duration")
	}
}

func TestBuildRuleRejectsUnknownFault(t *testing.T) {
	if _, err := BuildRule(RuleSpec{Faults: []FaultSpec{{Type: "explode"}}}); err == nil {
		t.Fatal("expected error for unknown fault type")
	}
}

func TestBuildRuleRejectsOutOfRangeProbability(t *testing.T) {
	_, err := BuildRule(RuleSpec{
		Counter: CounterSpec{
			Type: "probability",
			P:    2.0,
		},
	})
	if err == nil {
		t.Fatal("expected error for p=2.0, got nil (did it panic instead?)")
	}
}

func TestBuildRuleRejectsTooLargeLatency(t *testing.T) {
	_, err := BuildRule(RuleSpec{
		Faults: []FaultSpec{{Type: "latency", Duration: "10m"}},
	})
	if err == nil {
		t.Fatal("expected error for latency > maxFaultLatency, got nil")
	}
}

func TestBuildRuleReportsAllErrors(t *testing.T) {
	_, err := BuildRule(RuleSpec{
		Kinds:   []string{"not_a_kind"},
		Counter: CounterSpec{Type: "probability", P: -1},
		Faults:  []FaultSpec{{Type: "latency", Duration: "nope"}},
	})
	if err == nil {
		t.Fatal("expected aggregated error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"not_a_kind", "[0,1]", "nope"} {
		if !strings.Contains(msg, want) {
			t.Errorf("aggregated error missing %q; got: %v", want, msg)
		}
	}
}

func TestBuildRuleStagesRoundTrip(t *testing.T) {
	spec := RuleSpec{
		Name:  "flaky",
		Kinds: []string{"http_client"},
		Stages: []StageSpec{
			{Times: 2, Faults: []FaultSpec{{Type: "latency", Duration: "0s"}}},
			{Times: 0, Faults: []FaultSpec{{Type: "error", Message: "down"}}},
		},
	}
	r, err := BuildRule(spec)
	if err != nil {
		t.Fatalf("BuildRule: %v", err)
	}
	if r.staged == nil {
		t.Fatal("built rule should be staged")
	}
	// match 1-2: latency (nil error); match 3+: error.
	if _, f := r.staged.fire(); len(f) != 1 {
		t.Fatalf("stage 1 faults = %v", f)
	}
	r.staged.fire()                           // match 2
	if _, f := r.staged.fire(); len(f) != 1 { // match 3
		t.Fatal("stage 2 should have the error fault")
	}
}

func TestBuildRuleStagesValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		spec RuleSpec
		want string
	}{
		{
			"stages_with_counter",
			RuleSpec{Stages: []StageSpec{{Times: 0}}, Counter: CounterSpec{Type: "times", N: 1}},
			"cannot be combined with a counter",
		},
		{
			"stages_with_faults",
			RuleSpec{Stages: []StageSpec{{Times: 0}}, Faults: []FaultSpec{{Type: "conn_drop"}}},
			"cannot be combined with top-level faults",
		},
		{
			"nonfinal_zero",
			RuleSpec{Stages: []StageSpec{{Times: 0}, {Times: 1}}},
			"only the final stage",
		},
		{
			"bad_stage_fault",
			RuleSpec{Stages: []StageSpec{{Times: 0, Faults: []FaultSpec{{Type: "nope"}}}}},
			"unknown fault type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildRule(tt.spec)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestBuildFaultDisconnect(t *testing.T) {
	r, err := BuildRule(RuleSpec{
		Kinds:  []string{"net"},
		Faults: []FaultSpec{{Type: "disconnect"}},
	})
	if err != nil {
		t.Fatalf("BuildRule: %v", err)
	}
	if got := r.Info().Faults; len(got) != 1 || got[0] != fault.KindDisconnect {
		t.Fatalf("faults = %v, want [KindDisconnect]", got)
	}
}

func TestBuildFaultStream(t *testing.T) {
	tests := []struct {
		name string
		fs   FaultSpec
		kind fault.Kind
	}{
		{"slow_reader", FaultSpec{Type: "slow_reader", Rate: 1024}, fault.KindSlowReader},
		{"slow_writer", FaultSpec{Type: "slow_writer", Rate: 512}, fault.KindSlowWriter},
		{"truncate", FaultSpec{Type: "truncate", Limit: 8}, fault.KindTruncate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := BuildRule(RuleSpec{Kinds: []string{"io"}, Faults: []FaultSpec{tt.fs}})
			if err != nil {
				t.Fatalf("BuildRule: %v", err)
			}
			if got := r.Info().Faults; len(got) != 1 || got[0] != tt.kind {
				t.Fatalf("faults = %v, want [%v]", got, tt.kind)
			}
		})
	}
}

func TestBuildFaultStreamNegativeRejected(t *testing.T) {
	_, err := BuildRule(RuleSpec{Kinds: []string{"io"}, Faults: []FaultSpec{{Type: "slow_reader", Rate: -1}}})
	if err == nil || !strings.Contains(err.Error(), "rate") {
		t.Fatalf("err = %v, want a rate error", err)
	}
}
