package engine

import (
	"context"
	"regexp"
	"testing"
)

func TestMatchNameRegex(t *testing.T) {
	re := regexp.MustCompile(`^/api/v[0-9]+/users/.*$`)
	r := NewRule(MatchNameRegex(re))

	if !r.matches(context.Background(), Op{Name: "/api/v2/users/42"}) {
		t.Fatal("pattern should match /api/v2/users/42 (crosses slashes)")
	}
	if r.matches(context.Background(), Op{Name: "/api/v2/orders/42"}) {
		t.Fatal("pattern should not match /api/v2/orders/42")
	}
}

func TestMatchNameRegexNilNeverMatches(t *testing.T) {
	r := NewRule(MatchNameRegex(nil))
	if r.matches(context.Background(), Op{Name: "anything"}) {
		t.Fatal("a nil regexp should match nothing, not panic")
	}
}
