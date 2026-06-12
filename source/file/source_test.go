package file_test

import (
	"context"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/file"
)

func TestParseBuildsRuleSet(t *testing.T) {
	doc := []byte(`
meta:
  version: 1
rules:
  - name: slow-users
    kinds: [http_client]
    name_glob: /users/*
    counter:
      type: times
      n: 3
    faults:
      - type: latency
        duration: 50ms
      - type: error
        message: boom
`)
	rs, err := file.Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	if rs.Len() != 1 {
		t.Fatalf("Len = %d, want 1", rs.Len())
	}
	e := engine.New(engine.WithRuleSource(rs))
	if a := e.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient, Name: "/users/1"}); a == engine.Pass {
		t.Fatal("expected the loaded rule to fire")
	}
}

func TestParseRejectsDuplicateNames(t *testing.T) {
	_, err := file.Parse([]byte("rules:\n  - name: a\n  - name: a\n"))
	if err == nil {
		t.Fatal("expected duplicate-name error")
	}
}
