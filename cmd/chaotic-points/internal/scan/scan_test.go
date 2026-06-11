package scan

import (
	"path/filepath"
	"testing"
)

func TestScanFixture(t *testing.T) {
	dir, err := filepath.Abs("testdata/fixture")
	if err != nil {
		t.Fatal(err)
	}
	pts, err := Dir(dir, "./...")
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}

	// Collect literal names and count dynamic calls.
	names := map[string]bool{}
	dynamic := 0
	for _, p := range pts {
		if p.Dynamic {
			dynamic++
			continue
		}
		names[p.Name] = true
	}

	for _, want := range []string{"checkout.afterCommit", "order.beforeShip", "payment.retry"} {
		if !names[want] {
			t.Errorf("missing discovered point %q (got %v)", want, names)
		}
	}
	if dynamic != 1 {
		t.Errorf("dynamic call count = %d, want 1", dynamic)
	}
}
