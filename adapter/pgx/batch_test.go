package pgx

import (
	"errors"
	"testing"
)

func TestChaosBatchAllMethodsReturnError(t *testing.T) {
	want := errors.New("chaos")
	b := chaosBatch{err: want}

	if _, got := b.Exec(); !errors.Is(got, want) {
		t.Errorf("Exec err = %v, want %v", got, want)
	}
	if _, got := b.Query(); !errors.Is(got, want) {
		t.Errorf("Query err = %v, want %v", got, want)
	}
	if got := b.QueryRow().Scan(); !errors.Is(got, want) {
		t.Errorf("QueryRow.Scan err = %v, want %v", got, want)
	}
	if got := b.Close(); !errors.Is(got, want) {
		t.Errorf("Close err = %v, want %v", got, want)
	}
}
