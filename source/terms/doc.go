// Package terms parses a compact one-line chaos rule DSL into engine.RuleSpec /
// engine.Rule. It is the terse text form behind env-var and HTTP activation;
// because it decodes to RuleSpec, every parsed rule reuses engine.BuildRule
// validation and the engine.LintSpecs blast-radius check.
//
// Grammar (v1 — single term per rule):
//
//	ruleset  = rule { ";" rule }
//	rule     = [ name ":" ] [ selector { "," selector } "=" ] term
//	selector = "kind(" KIND ")" | "name(" GLOB ")" | "attr(" key "=" value ")"
//	term     = [ mode ] action [ "(" args ")" ]
//	mode     = INT "*"   (-> engine.Times)  |  FLOAT "%"  (-> engine.Probability, seed 0)
//	action   = latency | jitter | error | panic | conndrop | off
//
// Action mapping:
//
//	latency(200ms)        -> fault.Latency(200ms)
//	jitter(10ms,200ms)    -> fault.Jittered(10ms,200ms)
//	error("boom")         -> fault.Error(errors.New("boom"))
//	panic("kaboom")       -> fault.Panic("kaboom")
//	conndrop              -> fault.ConnDrop()
//	off                   -> rule present but inert (no faults)
//
// Example:
//
//	rules, err := terms.Compile(`flaky: kind(http_client),name(/users/*)=2*latency(200ms)`)
//
// Note for gofail users: there is no return(v). chaotic injects faults, it does
// not substitute return values, so gofail's return is renamed error here.
//
// Limits: the DSL emits only the serializable RuleSpec subset — no MatchPredicate
// and no typed errors.
//
// Staged faults: "term -> term -> ..." compiles to engine.WithStages. A leading
// "N*" sets a stage's match count; the final stage may omit the count to fire
// forever, e.g. 2*latency(200ms)->error("boom").
package terms
