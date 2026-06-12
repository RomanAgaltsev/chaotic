// Package golden records a chaos fault fire-sequence from one test run and
// replays it deterministically in another, to turn a flaky CI failure into a
// reproducible local one.
//
// Record installs a recording observer. On test cleanup it writes
// testdata/<name>.golden (JSON lines) when run with -chaos-update-golden (or
// CHAOS_UPDATE_GOLDEN=1). Replay reads that file, installs rules that reproduce
// the recorded fire/skip sequence via engine.Sequence and asserts the run does
// not diverge.
//
// Fidelity: latency/jittered fires replay with their recorded duration.
// Short-circuiting fires (error, panic, connection drop), whose kind the
// observer cannot see, replay as a generic fault.Error(ErrReplay). Replay
// assumes a deterministic (effectively single-goroutine) evaluation order and
// is faithful when each original rule's selector is distinguishable by Op.Kind.
//
// Usage — capture once, replay forever:
//
//	func TestCheckoutResilience(t *testing.T) {
//		// Record: run `go test -chaos-update-golden` to (re)write testdata/checkout.golden.
//		eng := chaostest.New(t, golden.Record(t, "checkout"))
//		// ... configure your real chaos rules, run the scenario ...
//		_ = eng
//	}
//
//	func TestCheckoutReplay(t *testing.T) {
//		// Replay: deterministically reproduce the recorded fire-sequence and assert
//		// the run does not diverge from testdata/checkout.golden.
//		eng := chaostest.New(t, golden.Replay(t, "checkout"))
//		// ... run the same scenario; divergence fails the test on cleanup ...
//		_ = eng
//	}
package golden

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// goldenEvent is one serialized engine decision. Fired distinguishes a fire
// (Fired=true, FaultKind/LatencyNS set) from a skip (Fired=false, Reason set).
type goldenEvent struct {
	Fired     bool              `json:"fired"`
	Rule      string            `json:"rule"`
	Kind      int               `json:"kind"`
	Name      string            `json:"name,omitempty"`
	Method    string            `json:"method,omitempty"`
	Attrs     map[string]string `json:"attrs,omitempty"`
	Reason    string            `json:"reason,omitempty"`
	FaultKind int               `json:"fault_kind,omitempty"`
	LatencyNS int64             `json:"latency_ns,omitempty"`
}

// writeEvents serializes events as JSON Lines (one event per line).
func writeEvents(w io.Writer, events []goldenEvent) error {
	bw := bufio.NewWriter(w)
	enc := json.NewEncoder(bw)
	for i := range events {
		if err := enc.Encode(events[i]); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// readEvents parses JSON Lines produced by writeEvents.
func readEvents(r io.Reader) ([]goldenEvent, error) {
	var out []goldenEvent
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev goldenEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, sc.Err()
}

// recorder is an engine.Observer + RichObserver that captures the ordered event
// stream. RuleFired appends a fired event. FaultInjected (which follows on the
// same goroutine during Before) annotates the most recent un-annotated fired
// event with the fault kind and latency.
type recorder struct {
	mu     sync.Mutex
	events []goldenEvent
}

func (r *recorder) RuleFired(name string, op engine.Op, _ engine.Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, goldenEvent{
		Fired:  true,
		Rule:   name,
		Kind:   int(op.Kind),
		Name:   op.Name,
		Method: op.Method,
		Attrs:  op.Attrs,
	})
}

func (r *recorder) RuleSkipped(name string, op engine.Op, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, goldenEvent{
		Rule:   name,
		Kind:   int(op.Kind),
		Name:   op.Name,
		Method: op.Method,
		Attrs:  op.Attrs,
		Reason: reason,
	})
}

// FaultInjected matches engine.RichObserver: (context.Context, engine.FaultEvent).
// It runs during Before on the same goroutine that produced the preceding
// RuleFired, so it annotates the most recent un-annotated fired event for the
// rule with the fault kind and latency.
func (r *recorder) FaultInjected(_ context.Context, ev engine.FaultEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].Fired && r.events[i].Rule == ev.Rule && r.events[i].FaultKind == 0 {
			r.events[i].FaultKind = int(ev.FaultKind)
			r.events[i].LatencyNS = int64(ev.Latency)
			return
		}
	}
}

func (r *recorder) snapshot() []goldenEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]goldenEvent(nil), r.events...)
}

// ErrReplay is the fault error injected for replayed short-circuiting fires
// whose original kind the observer could not see (error, panic, conn drop).
var ErrReplay = errors.New("chaotic/golden: replayed fault")

// buildReplayRules reconstructs one rule per recorded rule name. Each rule is
// selected by the distinct Op.Kinds the rule was evaluated against, fires via a
// Sequence mask built from that rule's ordered fire/skip decisions, and injects
// a fault reconstructed from the first fired event's kind (latency faithfully,
// everything else as ErrReplay). Rules with an empty name are ignored.
func buildReplayRules(events []goldenEvent) []engine.Rule {
	type acc struct {
		mask     []bool
		kinds    map[int]struct{}
		order    []int
		faultSet bool
		latency  time.Duration
		isErr    bool
	}
	byRule := map[string]*acc{}
	var ruleOrder []string
	for _, ev := range events {
		if ev.Rule == "" {
			continue
		}
		a := byRule[ev.Rule]
		if a == nil {
			a = &acc{kinds: map[int]struct{}{}}
			byRule[ev.Rule] = a
			ruleOrder = append(ruleOrder, ev.Rule)
		}
		a.mask = append(a.mask, ev.Fired)
		if _, ok := a.kinds[ev.Kind]; !ok {
			a.kinds[ev.Kind] = struct{}{}
			a.order = append(a.order, ev.Kind)
		}
		if ev.Fired && !a.faultSet {
			a.faultSet = true
			if ev.FaultKind == int(fault.KindLatency) || ev.FaultKind == int(fault.KindJittered) {
				a.latency = time.Duration(ev.LatencyNS)
			} else {
				a.isErr = true
			}
		}
	}
	var rules []engine.Rule
	for _, name := range ruleOrder {
		a := byRule[name]
		kinds := make([]engine.Kind, 0, len(a.order))
		for _, k := range a.order {
			kinds = append(kinds, engine.Kind(k))
		}
		var f fault.Fault
		if a.isErr || !a.faultSet {
			f = fault.Error(ErrReplay)
		} else {
			f = fault.Latency(a.latency)
		}
		rules = append(rules, engine.NewRule(
			engine.MatchKind(kinds...),
			engine.Sequence(a.mask),
			engine.WithFault(f),
		).Named(name))
	}
	return rules
}

// firedNames returns the ordered rule names of the fired events.
func firedNames(events []goldenEvent) []string {
	var out []string
	for _, ev := range events {
		if ev.Fired {
			out = append(out, ev.Rule)
		}
	}
	return out
}

// diffSequences returns "" if a and b are identical, else a short description.
func diffSequences(a, b []string) string {
	if len(a) != len(b) {
		return fmt.Sprintf("fired count %d != %d\n  golden: %s\n  replay: %s",
			len(a), len(b), strings.Join(a, ","), strings.Join(b, ","))
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Sprintf("fired[%d] %q != %q\n  golden: %s\n  replay: %s",
				i, a[i], b[i], strings.Join(a, ","), strings.Join(b, ","))
		}
	}
	return ""
}

var updateGolden = flag.Bool("chaos-update-golden", false,
	"rewrite chaostest/golden testdata files instead of asserting against them")

func updateEnabled() bool {
	if updateGolden != nil && *updateGolden {
		return true
	}
	return os.Getenv("CHAOS_UPDATE_GOLDEN") == "1"
}

func goldenPath(name string) string {
	return filepath.Join("testdata", name+".golden")
}

func writeGolden(path string, events []goldenEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	f, err := os.Create(path) //nolint:gosec // path is a fixed testdata/<name>.golden built by goldenPath
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return writeEvents(f, events)
}

func readGolden(path string) ([]goldenEvent, error) {
	f, err := os.Open(path) //nolint:gosec // path is a fixed testdata/<name>.golden built by goldenPath
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return readEvents(f)
}

// Record installs a recording observer on the engine and registers a t.Cleanup
// that writes testdata/<name>.golden when run with -chaos-update-golden (or
// CHAOS_UPDATE_GOLDEN=1). Without the update flag the cleanup does not write —
// Record is how you (re)generate a golden, not assert one.
//
// Record occupies the engine's single observer slot for the test; a test that
// also needs a production observer must fan in manually.
func Record(t testing.TB, name string) engine.Option {
	t.Helper()
	rec := &recorder{}
	t.Cleanup(func() {
		if !updateEnabled() {
			return
		}
		if err := writeGolden(goldenPath(name), rec.snapshot()); err != nil {
			t.Fatalf("golden: write %s: %v", name, err)
		}
	})
	return engine.WithObserver(rec)
}

// Replay reads testdata/<name>.golden, installs rules that reproduce the
// recorded fire/skip sequence (via engine.Sequence), and on t.Cleanup asserts
// the run's fired sequence equals the golden (t.Errorf on divergence).
//
// Like Record, it occupies the engine's single observer slot. Replay should be
// the sole rule source for the engine under test.
func Replay(t testing.TB, name string) engine.Option {
	t.Helper()
	events, err := readGolden(goldenPath(name))
	if err != nil {
		t.Fatalf("golden: read %s: %v (run with -chaos-update-golden to create it)", name, err)
	}
	rec := &recorder{}
	t.Cleanup(func() {
		if diff := diffSequences(firedNames(events), firedNames(rec.snapshot())); diff != "" {
			t.Errorf("golden %s: replay diverged:\n%s", name, diff)
		}
	})
	rules := buildReplayRules(events)
	return func(e *engine.Engine) {
		engine.WithRuleSource(engine.NewRuleSet(rules))(e)
		engine.WithObserver(rec)(e)
	}
}
