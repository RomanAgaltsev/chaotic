package terms

import (
	"context"
	"errors"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestSplitTopRespectsParens(t *testing.T) {
	got := splitTop("a,attr(k=v),jitter(1ms,2ms)", ',')
	want := []string{"a", "attr(k=v)", "jitter(1ms,2ms)"}
	if len(got) != len(want) {
		t.Fatalf("splitTop = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitTop[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIndexTopIgnoresParenContent(t *testing.T) {
	// The top-level '=' separates selectors from the term; the '=' inside
	// attr(k=v) must be ignored.
	s := "kind(http_client),attr(k=v)=error(\"boom\")"
	i := indexTop(s, '=')
	if i < 0 || s[i] != '=' {
		t.Fatalf("indexTop found no top-level '=' in %q", s)
	}
	if got := s[:i]; got != "kind(http_client),attr(k=v)" {
		t.Fatalf("selector part = %q", got)
	}
}

func TestSplitCall(t *testing.T) {
	name, args, err := splitCall("latency(200ms)")
	if err != nil || name != "latency" || args != "200ms" {
		t.Fatalf("splitCall = (%q,%q,%v)", name, args, err)
	}
	name, args, err = splitCall("conndrop")
	if err != nil || name != "conndrop" || args != "" {
		t.Fatalf("bare token splitCall = (%q,%q,%v)", name, args, err)
	}
	if _, _, err := splitCall("latency(200ms"); err == nil {
		t.Fatal("missing ) should error")
	}
}

func TestParseFullRule(t *testing.T) {
	specs, err := Parse(`flaky: kind(http_client),name(/users/*),attr(tier=gold)=2*latency(200ms)`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want 1", len(specs))
	}
	s := specs[0]
	if s.Name != "flaky" {
		t.Fatalf("name = %q", s.Name)
	}
	if len(s.Kinds) != 1 || s.Kinds[0] != "http_client" {
		t.Fatalf("kinds = %v", s.Kinds)
	}
	if s.NameGlob != "/users/*" {
		t.Fatalf("name_glob = %q", s.NameGlob)
	}
	if s.Attrs["tier"] != "gold" {
		t.Fatalf("attrs = %v", s.Attrs)
	}
	if s.Counter.Type != "times" || s.Counter.N != 2 {
		t.Fatalf("counter = %+v", s.Counter)
	}
	if len(s.Faults) != 1 || s.Faults[0].Type != "latency" || s.Faults[0].Duration != "200ms" {
		t.Fatalf("faults = %+v", s.Faults)
	}
}

func TestParseActionsAndModes(t *testing.T) {
	cases := map[string]func(engine.RuleSpec) bool{
		`error("boom")`:   func(s engine.RuleSpec) bool { return s.Faults[0].Type == "error" && s.Faults[0].Message == "boom" },
		`panic("kaboom")`: func(s engine.RuleSpec) bool { return s.Faults[0].Type == "panic" && s.Faults[0].Value == "kaboom" },
		`conndrop`:        func(s engine.RuleSpec) bool { return s.Faults[0].Type == "conn_drop" },
		`jitter(10ms,200ms)`: func(s engine.RuleSpec) bool {
			return s.Faults[0].Type == "jittered" && s.Faults[0].Min == "10ms" && s.Faults[0].Max == "200ms"
		},
		`40%error("x")`: func(s engine.RuleSpec) bool { return s.Counter.Type == "probability" && s.Counter.P == 0.4 },
		`off`:           func(s engine.RuleSpec) bool { return len(s.Faults) == 0 },
	}
	for in, ok := range cases {
		specs, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if !ok(specs[0]) {
			t.Fatalf("Parse(%q) produced unexpected spec %+v", in, specs[0])
		}
	}
}

func TestParseMultipleRules(t *testing.T) {
	specs, err := Parse(`a: error("x"); b: conndrop`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(specs) != 2 || specs[0].Name != "a" || specs[1].Name != "b" {
		t.Fatalf("specs = %+v", specs)
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{
		``,                        // empty
		`kind(http_client)=zap()`, // unknown action
		`wat(x)=error("y")`,       // unknown selector
		`=error("y")`,             // empty selector list before '='
	} {
		if _, err := Parse(in); err == nil {
			t.Fatalf("Parse(%q) = nil error, want error", in)
		}
	}
}

func TestParseRejectsChaining(t *testing.T) {
	_, err := Parse(`2*latency(200ms)->error("boom")`)
	if err == nil {
		t.Fatal("Parse should reject '->' chaining in v1")
	}
}

func TestCompileProducesUsableRule(t *testing.T) {
	rules, err := Compile(`flaky: kind(http_client)=2*error("upstream 503")`)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}

	eng := engine.New().AddRule(rules[0])
	ctx := context.Background()
	op := engine.Op{Kind: engine.OpHTTPClient, Name: "/x"}

	// Times(2): the first two evals fire, the third does not.
	for i := 1; i <= 3; i++ {
		act := eng.Eval(ctx, op)
		err := act.Before(ctx)
		switch {
		case i <= 2 && err == nil:
			t.Fatalf("eval %d: want injected error, got nil", i)
		case i == 3 && err != nil:
			t.Fatalf("eval %d: want nil after Times(2) exhausted, got %v", i, err)
		}
	}
	_ = errors.New
	_ = fault.ErrConnDrop
}

func TestCompileRejectsInvalidSpec(t *testing.T) {
	// Unknown kind is caught by BuildRule (not by Parse, which is structural).
	if _, err := Compile(`kind(not_a_kind)=error("x")`); err == nil {
		t.Fatal("Compile should reject an unknown kind via BuildRule")
	}
	// Bad duration is caught by BuildRule.
	if _, err := Compile(`latency(not_a_duration)`); err == nil {
		t.Fatal("Compile should reject a bad duration via BuildRule")
	}
}
