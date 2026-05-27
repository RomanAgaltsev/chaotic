package engine

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/ag4r/chaotic/fault"
)

func TestMatchKindMatchesAnyOfTheListedKinds(t *testing.T) {
	r := NewRule(MatchKind(OpHTTPClient, OpSQL))
	if !r.matches(context.Background(), Op{Kind: OpHTTPClient}) {
		t.Fatal("expected OpHTTPClient to match")
	}
	if !r.matches(context.Background(), Op{Kind: OpSQL}) {
		t.Fatal("expected OpSQL to match")
	}
	if r.matches(context.Background(), Op{Kind: OpGRPCClient}) {
		t.Fatal("expected OpGRPCClient to not match")
	}
}

func TestMatchNameGlobsLastSegment(t *testing.T) {
	r := NewRule(MatchName("/users/*"))
	cases := map[string]bool{
		"/users/123":       true,
		"/users/abc":       true,
		"/users/123/posts": false,
		"/other/123":       false,
	}
	for name, want := range cases {
		got := r.matches(context.Background(), Op{Name: name})
		if got != want {
			t.Errorf("MatchName(\"/users/*\") on %q: got %v want %v", name, got, want)
		}
	}
}

func TestMatchAttrRequiresExactValue(t *testing.T) {
	r := NewRule(MatchAttr("host", "api.example.com"))
	if !r.matches(context.Background(), Op{Attrs: map[string]string{"host": "api.example.com"}}) {
		t.Fatal("expected match")
	}
	if r.matches(context.Background(), Op{Attrs: map[string]string{"host": "other"}}) {
		t.Fatal("expected no match")
	}
	if r.matches(context.Background(), Op{}) {
		t.Fatal("expected no match for nil Attrs")
	}
}

func TestMatchPredicateUserDefined(t *testing.T) {
	r := NewRule(MatchPredicate(func(_ context.Context, op Op) bool {
		return op.Method == http.MethodPost
	}))
	if !r.matches(context.Background(), Op{Method: "POST"}) {
		t.Fatal("expected POST to match")
	}
	if r.matches(context.Background(), Op{Method: "GET"}) {
		t.Fatal("expected GET not to match")
	}
}

func TestEmptyRuleMatchesEverything(t *testing.T) {
	r := NewRule()
	if !r.matches(context.Background(), Op{Kind: OpHTTPClient}) {
		t.Fatal("expected empty rule to match any Op")
	}
}

func TestSelectorsAreANDed(t *testing.T) {
	r := NewRule(
		MatchKind(OpHTTPClient),
		MatchAttr("host", "x"),
	)
	if !r.matches(context.Background(), Op{Kind: OpHTTPClient, Attrs: map[string]string{"host": "x"}}) {
		t.Fatal("expected match when both selectors satisfied")
	}
	if r.matches(context.Background(), Op{Kind: OpHTTPClient}) {
		t.Fatal("expected no match when attrs missing")
	}
	if r.matches(context.Background(), Op{Kind: OpSQL, Attrs: map[string]string{"host": "x"}}) {
		t.Fatal("expected no match when kind wrong")
	}
}

func TestAlwaysFiresEveryTime(t *testing.T) {
	r := NewRule(Always())
	for i := range 5 {
		if !r.counter.shouldFire() {
			t.Fatalf("Always returned false on iteration %d", i)
		}
	}
}

func TestTimesFiresFirstN(t *testing.T) {
	r := NewRule(Times(3))
	got := []bool{}
	for range 6 {
		got = append(got, r.counter.shouldFire())
	}
	want := []bool{true, true, true, false, false, false}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("iteration %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestRangeFiresInWindow(t *testing.T) {
	r := NewRule(Range(2, 4))
	got := []bool{}
	for range 6 {
		got = append(got, r.counter.shouldFire())
	}
	want := []bool{false, true, true, true, false, false}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("iteration %d (1-indexed %d): got %v want %v", i, i+1, got[i], want[i])
		}
	}
}

func TestProbabilityIsDeterministic(t *testing.T) {
	r1 := NewRule(Probability(0.5, 42))
	r2 := NewRule(Probability(0.5, 42))
	for i := range 20 {
		a := r1.counter.shouldFire()
		b := r2.counter.shouldFire()
		if a != b {
			t.Fatalf("iteration %d: same seed produced different results (%v vs %v)", i, a, b)
		}
	}
}

func TestProbabilityZeroNeverFires(t *testing.T) {
	r := NewRule(Probability(0.0, 1))
	for i := range 50 {
		if r.counter.shouldFire() {
			t.Fatalf("p=0 fired on iteration %d", i)
		}
	}
}

func TestProbabilityOneAlwaysFires(t *testing.T) {
	r := NewRule(Probability(1.0, 1))
	for i := range 50 {
		if !r.counter.shouldFire() {
			t.Fatalf("p=1 did not fire on iteration %d", i)
		}
	}
}

func TestWithFaultAttachesSingleFault(t *testing.T) {
	target := errors.New("boom")
	r := NewRule(WithFault(fault.Error(target)))
	if len(r.faults) != 1 {
		t.Fatalf("got %d faults, want 1", len(r.faults))
	}
}

func TestWithFaultsAttachesInOrder(t *testing.T) {
	target := errors.New("boom")
	r := NewRule(WithFaults(fault.Latency(0), fault.Error(target)))
	if len(r.faults) != 2 {
		t.Fatalf("got %d faults, want 2", len(r.faults))
	}
}

func TestNamedReturnsCopyWithName(t *testing.T) {
	orig := NewRule(Times(2))
	named := orig.Named("test-rule")
	if orig.name != "" {
		t.Fatalf("original mutated: name=%q", orig.name)
	}
	if named.name != "test-rule" {
		t.Fatalf("named.name = %q, want %q", named.name, "test-rule")
	}
	// Counter is shared — calling shouldFire on one affects the other.
	if !named.counter.shouldFire() {
		t.Fatal("named first call should fire")
	}
	if !orig.counter.shouldFire() {
		t.Fatal("orig second call should fire (Times(2))")
	}
	if orig.counter.shouldFire() {
		t.Fatal("orig third call should not fire")
	}
}

func TestSliceRuleSetSnapshot(t *testing.T) {
	rs := newSliceRuleSet([]Rule{NewRule().Named("a"), NewRule().Named("b")})
	if rs.Len() != 2 {
		t.Fatalf("Len = %d, want 2", rs.Len())
	}
	snap := rs.Snapshot()
	if len(snap) != 2 || snap[0].Name() != "a" || snap[1].Name() != "b" {
		t.Fatalf("Snapshot = %+v, want [{a}:{b}]", snap)
	}
	// Mutating the snapshot does not affect the ruleset (test by adding new rule).
	snap[0] = NewRule().Named("c")
	again := rs.Snapshot()
	if again[0].Name() != "a" {
		t.Fatalf("Snapshot mutation leaked back: got %q", again[0].Name())
	}
}
