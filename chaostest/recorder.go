package chaostest

import (
	"sync"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// Event is one observed engine decision. Fired distinguishes a RuleFired event
// (Fired=true, Reason="") from a RuleSkipped event (Fired=false, Reason set).
type Event struct {
	Fired  bool
	Rule   string
	Op     engine.Op
	Reason string
}

// Recorder is an engine.Observer that records events for assertions. Attach it
// with engine.WithObserver. Safe for concurrent use. Methods run on the request
// path. A Recorder is an instance - never share one across engines you want to
// assert independently.
type Recorder struct {
	mu     sync.Mutex
	events []Event
}

// NewRecorder returns an empty Recorder.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// RuleFired adds an event of fired rule to the recorder.
func (r *Recorder) RuleFired(name string, op engine.Op, _ engine.Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, Event{Fired: true, Rule: name, Op: op})
}

// RuleSkipped adds an event of skipped rule to the recorder.
func (r *Recorder) RuleSkipped(name string, op engine.Op, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, Event{Rule: name, Op: op, Reason: reason})
}

// Events returns a copy of all recorded events in order.
func (r *Recorder) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Event(nil), r.events...)
}

// Fired returns the fired events for ruleName (or all fired events if "").
func (r *Recorder) Fired(ruleName string) []Event { return r.filter(true, ruleName) }

// Skipped returns the skipped events for ruleName (or all skips if "").
func (r *Recorder) Skipped(ruleName string) []Event { return r.filter(false, ruleName) }

func (r *Recorder) filter(fired bool, name string) []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Event
	for _, e := range r.events {
		if e.Fired == fired && (name == "" || e.Rule == name) {
			out = append(out, e)
		}
	}
	return out
}
