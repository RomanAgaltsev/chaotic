package env_test

import (
	"context"
	"errors"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/env"
	"github.com/RomanAgaltsev/chaotic/source/terms"
)

func TestFromEnvParsesRules(t *testing.T) {
	t.Setenv("CHAOTIC_RULES", `kind(http_client)=error("boom")`)
	rs, err := env.FromEnv("CHAOTIC_RULES")
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	eng := engine.New(engine.WithRuleSource(rs))
	act := eng.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient, Name: "/x"})
	if act.Before(context.Background()) == nil {
		t.Fatal("expected the env rule to fire")
	}
}

func TestFromEnvEmptyIsNoOp(t *testing.T) {
	rs, err := env.FromEnv("CHAOTIC_RULES_UNSET_XYZ")
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	eng := engine.New(engine.WithRuleSource(rs))
	if eng.Enabled() {
		t.Fatal("an unset var must yield a disabled engine")
	}
}

func TestFromEnvMalformedErrors(t *testing.T) {
	t.Setenv("CHAOTIC_RULES", `kind(http_client)=bogus(`)
	if _, err := env.FromEnv("CHAOTIC_RULES"); err == nil {
		t.Fatal("expected a parse error")
	}
}

func TestFromEnvDefaultName(t *testing.T) {
	t.Setenv("CHAOTIC_RULES", `kind(sql)=conndrop`)
	rs, err := env.FromEnv("") // "" => default CHAOTIC_RULES
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if engine.New(engine.WithRuleSource(rs)).Enabled() == false {
		t.Fatal("default var name should have loaded the rule")
	}
}

func TestFromEnvLintRejectFailsOnHazard(t *testing.T) {
	t.Setenv("CHAOTIC_RULES", `wipeout: panic("boom")`)

	_, err := env.FromEnv("", terms.WithLint(engine.LintReject))
	if !errors.Is(err, engine.ErrLintRejected) {
		t.Fatalf("err = %v, want ErrLintRejected", err)
	}
}

func TestFromEnvLintOffIsDefault(t *testing.T) {
	t.Setenv("CHAOTIC_RULES", `wipeout: panic("boom")`)

	rs, err := env.FromEnv("")
	if err != nil {
		t.Fatalf("FromEnv without options rejected a hazard: %v", err)
	}
	if rs.Len() != 1 {
		t.Fatalf("got %d rules, want 1", rs.Len())
	}
}
