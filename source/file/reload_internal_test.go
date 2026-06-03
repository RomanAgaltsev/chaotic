package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ag4r/chaotic/engine"
)

func TestReloadReturnsErrorBadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yml")
	// out-of-range probability: BuildRule now errors (does not panic)
	if err := os.WriteFile(path, []byte("rules:\n  - name: bad\n    counter:\n      type: probability\n      p: 2.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	eng := engine.New()
	if _, err := reload(path, eng); err == nil {
		t.Fatal("expected reload error for out-of-range probability, got nil")
	}
}
