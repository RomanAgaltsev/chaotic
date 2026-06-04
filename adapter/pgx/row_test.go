package pgx

import (
	"errors"
	"testing"
)

func TestChaosRowScanReturnsError(t *testing.T) {
	want := errors.New("chaos")
	r := chaosRow{err: want}
	if got := r.Scan(); !errors.Is(got, want) {
		t.Fatalf("Scan() = %v, want errors.Is(%v) == true", got, want)
	}
}
