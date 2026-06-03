package main

import "testing"

func TestPoisonedConnIsRetried(t *testing.T) {
	opens, err := run()
	if err != nil {
		t.Fatalf("exec did not recover from a poisoned conn: %v", err)
	}
	if opens < 2 {
		t.Fatalf("driver opened %d conns, want >= 2 (a fresh conn after the poison)", opens)
	}
}
