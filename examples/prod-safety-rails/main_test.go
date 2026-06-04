//go:build !chaos_off

package main

import "testing"

func TestFailureBudgetBoundsInjection(t *testing.T) {
	injected, total := run()
	if injected == 0 {
		t.Fatal("no calls were faulted; the chaos rule never engaged")
	}
	if injected == total {
		t.Fatalf("all %d calls were faulted; the failure budget never tripped", total)
	}
}
