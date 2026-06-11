package scan

import (
	"path/filepath"
	"testing"

	"github.com/ag4r/chaotic/engine"
)

func TestGate(t *testing.T) {
	points := []Point{
		{Name: "checkout.afterCommit"},
		{Name: "order.beforeShip"},
		{Dynamic: true},
	}
	specs := []engine.RuleSpec{
		{Name: "ok-literal", Kinds: []string{"explicit"}, NameGlob: "checkout.afterCommit"},
		{Name: "typo", Kinds: []string{"explicit"}, NameGlob: "checkout.afterCommt"},
		{Name: "glob-hit", Kinds: []string{"explicit"}, NameGlob: "checkout.*"},
		{Name: "glob-miss", Kinds: []string{"explicit"}, NameGlob: "nope.*"},
		{Name: "other-kind", Kinds: []string{"http_client"}, NameGlob: "ignored"},
	}

	fs := Gate(points, specs, false)
	if len(fs) != 2 {
		t.Fatalf("findings = %d (%+v), want 2 (typo error + glob-miss warning)", len(fs), fs)
	}
	byRule := map[string]Finding{}
	for _, f := range fs {
		byRule[f.Rule] = f
	}
	if byRule["typo"].Level != "error" {
		t.Errorf("typo level = %q, want error", byRule["typo"].Level)
	}
	if byRule["glob-miss"].Level != "warning" {
		t.Errorf("glob-miss level = %q, want warning", byRule["glob-miss"].Level)
	}
	if _, ok := byRule["other-kind"]; ok {
		t.Error("non-explicit rule should be ignored")
	}

	// --strict promotes the glob miss to an error.
	strict := Gate(points, specs, true)
	for _, f := range strict {
		if f.Rule == "glob-miss" && f.Level != "error" {
			t.Errorf("glob-miss under strict = %q, want error", f.Level)
		}
	}
}

func TestScanThenGateFixture(t *testing.T) {
	dir, _ := filepath.Abs("testdata/fixture")
	pts, err := Dir(dir, "./...")
	if err != nil {
		t.Fatalf("ScanDir: %v", err)
	}
	specs := []engine.RuleSpec{
		{Name: "good", Kinds: []string{"explicit"}, NameGlob: "checkout.afterCommit"},
		{Name: "typo", Kinds: []string{"explicit"}, NameGlob: "checkout.afterCommt"},
	}
	fs := Gate(pts, specs, false)
	if len(fs) != 1 || fs[0].Rule != "typo" || fs[0].Level != "error" {
		t.Fatalf("findings = %+v, want one error for rule \"typo\"", fs)
	}
}
