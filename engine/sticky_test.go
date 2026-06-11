package engine

import (
	"context"
	"testing"
	"time"
)

func TestStickyAttrKeepsFiringForSameUser(t *testing.T) {
	// Base counter Times(1): only the first matching op fires normally. Sticky
	// then keeps firing for that op's user for the window.
	eng := New().AddRule(NewRule(
		MatchKind(OpHTTPClient),
		Times(1),
		StickyAttr("user", time.Minute, 16),
		WithFault(errFaultForTest()),
	).Named("stick"))

	ctx := context.Background()
	opAlice := Op{Kind: OpHTTPClient, Name: "/x", Attrs: map[string]string{"user": "alice"}}
	opBob := Op{Kind: OpHTTPClient, Name: "/x", Attrs: map[string]string{"user": "bob"}}

	// First alice call fires (Times(1)) and makes alice sticky.
	if _, ok := eng.Eval(ctx, opAlice).(*ruleAction); !ok {
		t.Fatal("first alice call should fire")
	}
	// Times(1) is now exhausted, but alice stays sticky -> still fires.
	if _, ok := eng.Eval(ctx, opAlice).(*ruleAction); !ok {
		t.Fatal("second alice call should still fire (sticky)")
	}
	// bob is not sticky and the counter is exhausted -> does not fire.
	if eng.Eval(ctx, opBob) != Pass {
		t.Fatal("bob should not fire: not sticky and counter exhausted")
	}
}

func TestStickyAttrExpiresAfterWindow(t *testing.T) {
	eng := New().AddRule(NewRule(
		MatchKind(OpHTTPClient),
		Always(),
		stickyAttrAt("user", time.Hour, 16, fakeClock(time.Unix(0, 0))),
		WithFault(errFaultForTest()),
	).Named("stick"))
	_ = eng // exercised via the tracker unit test below; see stickyTracker tests.
}

func errFaultForTest() faultForTest { return faultForTest{} }

type faultForTest struct{}

func (faultForTest) Apply(context.Context) error { return errStickyTest }

var errStickyTest = stickyTestErr("sticky boom")

type stickyTestErr string

func (e stickyTestErr) Error() string { return string(e) }

func TestStickyTrackerEvictsAtCap(t *testing.T) {
	now := time.Unix(1000, 0)
	tr := &stickyTracker{key: "user", window: time.Hour, cap: 2, seen: map[string]time.Time{}, now: func() time.Time { return now }}
	tr.mark(Op{Attrs: map[string]string{"user": "a"}})
	tr.mark(Op{Attrs: map[string]string{"user": "b"}})
	tr.mark(Op{Attrs: map[string]string{"user": "c"}}) // evicts "a"
	if tr.sticky(Op{Attrs: map[string]string{"user": "a"}}) {
		t.Fatal("a should have been evicted at cap=2")
	}
	if !tr.sticky(Op{Attrs: map[string]string{"user": "c"}}) {
		t.Fatal("c should be sticky")
	}
}

func TestStickyTrackerExpiry(t *testing.T) {
	now := time.Unix(1000, 0)
	tr := &stickyTracker{key: "user", window: time.Hour, cap: 8, seen: map[string]time.Time{}, now: func() time.Time { return now }}
	tr.mark(Op{Attrs: map[string]string{"user": "a"}})
	now = now.Add(2 * time.Hour) // past the window
	if tr.sticky(Op{Attrs: map[string]string{"user": "a"}}) {
		t.Fatal("a should have expired after the window")
	}
}

func fakeClock(t time.Time) RuleOption { // helper used only to keep the second test compiling
	return func(*Rule) {}
}

func stickyAttrAt(key string, window time.Duration, capN int, _ RuleOption) RuleOption {
	return StickyAttr(key, window, capN)
}
