// Package http exposes an admin endpoint to inspect and (optionally) install
// chaotic rules at runtime. GET returns the current YAML document. POST/PUT
// installs a new one when writable. Read-only by default.
package http

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/ag4r/chaotic/engine"
)

// Handler is an http.Handler bound to an engine.
type Handler struct {
	eng      *engine.Engine
	writable bool
	auth     func(token string) bool

	mu      sync.RWMutex
	current []byte // last-installed document, served on GET
}

// Option configures a Handler.
type Option func(*Handler)

// WithWritable enables POST/PUT rule installation. Default: read-only.
func WithWritable(w bool) Option {
	return func(h *Handler) {
		h.writable = w
	}
}

// WithAuth requires a bearer token accepted by check on every request.
func WithAuth(check func(token string) bool) Option {
	return func(h *Handler) { h.auth = check }
}

// New returns a Handler bound to eng.
func New(eng *engine.Engine, opts ...Option) *Handler {
	h := &Handler{eng: eng}
	for _, o := range opts {
		o(h)
	}
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.auth != nil && !h.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.mu.RLock()
		doc := h.current
		h.mu.RUnlock()
		w.Header().Set("Content-Type", "application/yaml")
		if doc == nil {
			_, _ = w.Write([]byte("rules: []\n"))
			return
		}
		_, _ = w.Write(doc)
	case http.MethodPost, http.MethodPut:
		if !h.writable {
			http.Error(w, "read-only", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rs, err := parseRuleSet(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.eng.ReplaceRules(rs)
		h.mu.Lock()
		h.current = body
		h.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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

func parseRuleSet(data []byte) (engine.RuleSet, error) {
	var doc ruleDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	rules := make([]engine.Rule, 0, len(doc.Rules))
	for i, spec := range doc.Rules {
		r, err := engine.BuildRule(spec)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i, err)
		}
		rules = append(rules, r)
	}
	return engine.NewRuleSet(rules), nil
}
