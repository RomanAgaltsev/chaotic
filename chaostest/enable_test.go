package chaostest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/RomanAgaltsev/chaotic/chaostest"
	"github.com/RomanAgaltsev/chaotic/engine"
)

// enableTB captures Fatalf and Cleanup so we can test Enable's failure and
// teardown paths without aborting the real test.
type enableTB struct {
	testing.TB
	fatal    []string
	cleanups []func()
}

func (f *enableTB) Helper() {}
func (f *enableTB) Fatalf(format string, args ...any) {
	f.fatal = append(f.fatal, fmt.Sprintf(format, args...))
}
func (f *enableTB) Cleanup(fn func()) { f.cleanups = append(f.cleanups, fn) }

func TestEnableAddsRulesAndNamesWork(t *testing.T) {
	eng := chaostest.New(t)
	names := chaostest.Enable(t, eng, `kind(http_client)=error("boom")`)
	if len(names) != 1 {
		t.Fatalf("names = %v, want exactly 1", names)
	}
	act := eng.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient})
	_ = act.Before(context.Background())
	chaostest.AssertHits(t, eng, names[0], 1)
}

func TestEnableFailsOnInvalidTerms(t *testing.T) {
	ft := &enableTB{}
	eng := engine.New()
	names := chaostest.Enable(ft, eng, `this is not valid terms`)
	if len(ft.fatal) == 0 {
		t.Fatal("Enable did not fail the test on invalid terms")
	}
	if names != nil {
		t.Fatalf("names = %v, want nil on failure", names)
	}
}

func TestEnableCleanupResetsEngine(t *testing.T) {
	ft := &enableTB{}
	eng := engine.New()
	chaostest.Enable(ft, eng, `kind(http_client)=error("boom")`)
	if !eng.Enabled() {
		t.Fatal("Enable did not add the rule")
	}
	for _, fn := range ft.cleanups {
		fn()
	}
	if eng.Enabled() {
		t.Fatal("registered cleanup did not reset the engine")
	}
}
