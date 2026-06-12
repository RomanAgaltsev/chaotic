// Package http exposes an admin endpoint to inspect and (optionally) install
// chaotic rules at runtime.
//
// Wholesale: GET / returns the current YAML document; POST/PUT / installs a new
// one (writable only). Per-rule (writable only): PUT /rules/{name} installs one
// rule from a source/terms string, DELETE /rules/{name} removes it. Read-only
// introspection: GET /rules lists names+hits, GET /rules/{name}/count returns a
// rule's hit count.
//
// The handler routes with an internal http.ServeMux rooted at "/", so mount it
// behind http.StripPrefix:
//
//	mux.Handle("/chaos/", http.StripPrefix("/chaos", chaoshttp.New(eng, chaoshttp.WithWritable(true))))
package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/terms"
)

// Handler is an http.Handler bound to an engine.
type Handler struct {
	eng      *engine.Engine
	writable bool
	auth     func(token string) bool
	mux      *http.ServeMux

	mu      sync.RWMutex
	order   []string                   // rule names in insertion order
	specs   map[string]engine.RuleSpec // authoritative current specs by name
	current []byte                     // YAML doc serialized from specs (served on GET /)
}

// Option configures a Handler.
type Option func(*Handler)

// WithWritable enables rule installation (wholesale and per-rule). Default: read-only.
func WithWritable(w bool) Option { return func(h *Handler) { h.writable = w } }

// WithAuth requires a bearer token accepted by check on every request.
func WithAuth(check func(token string) bool) Option { return func(h *Handler) { h.auth = check } }

// New returns a Handler bound to eng.
func New(eng *engine.Engine, opts ...Option) *Handler {
	h := &Handler{eng: eng, specs: map[string]engine.RuleSpec{}}
	for _, o := range opts {
		o(h)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.getWhole)
	mux.HandleFunc("POST /{$}", h.postWhole)
	mux.HandleFunc("PUT /{$}", h.postWhole)
	mux.HandleFunc("GET /rules", h.listRules)
	mux.HandleFunc("GET /rules/{name}/count", h.ruleCount)
	mux.HandleFunc("PUT /rules/{name}", h.putRule)
	mux.HandleFunc("DELETE /rules/{name}", h.deleteRule)
	h.mux = mux
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.auth != nil && !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) getWhole(w http.ResponseWriter, _ *http.Request) {
	h.mu.RLock()
	doc := h.current
	h.mu.RUnlock()
	w.Header().Set("Content-Type", "application/yaml")
	if doc == nil {
		_, _ = w.Write([]byte("rules: []\n"))
		return
	}
	_, _ = w.Write(doc)
}

func (h *Handler) postWhole(w http.ResponseWriter, r *http.Request) {
	if !h.writable {
		http.Error(w, "read-only", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var doc ruleDoc
	if err := yaml.Unmarshal(body, &doc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	order := make([]string, 0, len(doc.Rules))
	specs := make(map[string]engine.RuleSpec, len(doc.Rules))
	for i, spec := range doc.Rules {
		name := spec.Name
		if name == "" {
			name = fmt.Sprintf("rule-%d", i)
			spec.Name = name
		}
		if _, dup := specs[name]; !dup {
			order = append(order, name)
		}
		specs[name] = spec
	}
	h.mu.Lock()
	prevOrder, prevSpecs := h.order, h.specs
	h.order, h.specs = order, specs
	err = h.rebuildLocked()
	if err != nil {
		h.order, h.specs = prevOrder, prevSpecs // roll back on a bad document
	}
	h.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// rebuildLocked rebuilds the live rule set and the served YAML from h.specs in
// h.order. Caller must hold h.mu.
func (h *Handler) rebuildLocked() error {
	rules := make([]engine.Rule, 0, len(h.order))
	for _, name := range h.order {
		r, err := engine.BuildRule(h.specs[name])
		if err != nil {
			return fmt.Errorf("rule %q: %w", name, err)
		}
		rules = append(rules, r)
	}
	h.eng.ReplaceRules(engine.NewRuleSet(rules))
	doc := ruleDoc{Rules: make([]engine.RuleSpec, 0, len(h.order))}
	for _, name := range h.order {
		doc.Rules = append(doc.Rules, h.specs[name])
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	h.current = out
	return nil
}

func (h *Handler) authorized(r *http.Request) bool {
	const prefix = "Bearer "
	a := r.Header.Get("Authorization")
	if len(a) <= len(prefix) || a[:len(prefix)] != prefix {
		return false
	}
	return h.auth(a[len(prefix):])
}

type ruleDoc struct {
	Rules []engine.RuleSpec `yaml:"rules"`
}

func (h *Handler) listRules(w http.ResponseWriter, _ *http.Request) {
	hits := h.eng.AllHits()
	type item struct {
		Name string `json:"name"`
		Hits int    `json:"hits"`
	}
	h.mu.RLock()
	out := make([]item, 0, len(h.order))
	for _, name := range h.order {
		out = append(out, item{Name: name, Hits: hits[name]})
	}
	h.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) ruleCount(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	h.mu.RLock()
	_, ok := h.specs[name]
	h.mu.RUnlock()
	if !ok {
		http.Error(w, "no such rule", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	// G705 false positive: the only value written is the integer hit count via
	// %d; the request-derived rule name is used solely as a lookup key above and
	// is never emitted, so the response cannot carry user-controlled markup.
	_, _ = fmt.Fprintf(w, "%d\n", h.eng.Hits(name)) //nolint:gosec // integer-only output, no user-controlled data reaches the writer
}

func (h *Handler) putRule(w http.ResponseWriter, r *http.Request) {
	if !h.writable {
		http.Error(w, "read-only", http.StatusMethodNotAllowed)
		return
	}
	name := r.PathValue("name")
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	specs, err := terms.Parse(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(specs) != 1 {
		http.Error(w, "PUT /rules/{name} accepts exactly one rule", http.StatusBadRequest)
		return
	}
	spec := specs[0]
	spec.Name = name // the URL name is authoritative
	if _, err := engine.BuildRule(spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	if _, exists := h.specs[name]; !exists {
		h.order = append(h.order, name)
	}
	h.specs[name] = spec
	err = h.rebuildLocked()
	h.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteRule(w http.ResponseWriter, r *http.Request) {
	if !h.writable {
		http.Error(w, "read-only", http.StatusMethodNotAllowed)
		return
	}
	name := r.PathValue("name")
	h.mu.Lock()
	if _, ok := h.specs[name]; !ok {
		h.mu.Unlock()
		http.Error(w, "no such rule", http.StatusNotFound)
		return
	}
	delete(h.specs, name)
	h.order = removeName(h.order, name)
	err := h.rebuildLocked()
	h.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func removeName(names []string, name string) []string {
	out := names[:0]
	for _, n := range names {
		if n != name {
			out = append(out, n)
		}
	}
	return out
}
