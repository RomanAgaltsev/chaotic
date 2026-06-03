package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/ag4r/chaotic/fault"
)

// RuleSpec is the declarative, serializable form of a Rule. Rule sources parse
// config (YAML/JSON) into RuleSpec and call BuildRule. The struct tags are the
// on-disk/on-wire field names. MatchPredicate and typed error values cannot be
// serialized. Config rules support this declarative subset only.
type RuleSpec struct {
	Name     string            `yaml:"name" json:"name"`
	Kinds    []string          `yaml:"kinds" json:"kinds"`
	NameGlob string            `yaml:"name_glob" json:"name_glob"`
	Attrs    map[string]string `yaml:"attrs" json:"attrs"`
	Counter  CounterSpec       `yaml:"counter" json:"counter"`
	Faults   []FaultSpec       `yaml:"faults" json:"faults"`
}

// CounterSpec selects a counter. Type is "always", "times", "range", or
// "probability" (empty defaults to "always").
type CounterSpec struct {
	Type string  `yaml:"type" json:"type"`
	N    int     `yaml:"n" json:"n"`
	From int     `yaml:"from" json:"from"`
	To   int     `yaml:"to" json:"to"`
	P    float64 `yaml:"p" json:"p"`
	Seed int64   `yaml:"seed" json:"seed"`
}

// FaultSpec selects a fault. Type is "latency", "jittered", "error", "panic",
// or "conn_drop".
type FaultSpec struct {
	Type     string `yaml:"type" json:"type"`
	Duration string `yaml:"duration" json:"duration"` // latency
	Min      string `yaml:"min" json:"min"`           // jittered
	Max      string `yaml:"max" json:"max"`           // jittered
	Message  string `yaml:"message" json:"message"`   // error -> errors.New(Message)
	Value    string `yaml:"value" json:"value"`       // panic -> Panic(value)
}

var kindNames = map[string]Kind{
	"http_client": OpHTTPClient,
	"http_server": OpHTTPServer,
	"sql":         OpSQL,
	"grpc_client": OpGRPCClient,
	"grpc_server": OpGRPCServer,
}

// BuildRule converts a RuleSpec into a Rule, validating kinds, counter type,
// fault types, durations, and probability bounds. It is total: it never panics,
// and it aggregates every invalid field into a single errors.Join result so a
// caller (and the operator editing config) sees all problems at once.
func BuildRule(spec RuleSpec) (Rule, error) {
	var opts []RuleOption
	var errs []error

	if len(spec.Kinds) > 0 {
		kinds := make([]Kind, 0, len(spec.Kinds))
		for _, ks := range spec.Kinds {
			k, ok := kindNames[ks]
			if !ok {
				errs = append(errs, fmt.Errorf("chaotic: unknown kind %q", ks))
				continue
			}
			kinds = append(kinds, k)
		}
		if len(kinds) > 0 {
			opts = append(opts, MatchKind(kinds...))
		}
	}
	if spec.NameGlob != "" {
		opts = append(opts, MatchName(spec.NameGlob))
	}
	for k, v := range spec.Attrs {
		opts = append(opts, MatchAttr(k, v))
	}

	switch spec.Counter.Type {
	case "", "always":
		opts = append(opts, Always())
	case "times":
		opts = append(opts, Times(spec.Counter.N))
	case "range":
		opts = append(opts, Range(spec.Counter.From, spec.Counter.To))
	case "probability":
		if spec.Counter.P < 0 || spec.Counter.P > 1 {
			errs = append(errs, fmt.Errorf("chaotic: probability p=%v out of [0,1]", spec.Counter.P))
		} else {
			opts = append(opts, Probability(spec.Counter.P, spec.Counter.Seed))
		}
	default:
		errs = append(errs, fmt.Errorf("chaotic: unknown counter type %q", spec.Counter.Type))
	}

	faults := make([]fault.Fault, 0, len(spec.Faults))
	for _, fs := range spec.Faults {
		f, err := buildFault(fs)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		faults = append(faults, f)
	}
	if len(faults) > 0 {
		opts = append(opts, WithFaults(faults...))
	}

	if len(errs) > 0 {
		return Rule{}, errors.Join(errs...)
	}

	r := NewRule(opts...)
	if spec.Name != "" {
		r = r.Named(spec.Name)
	}
	return r, nil
}

// maxFaultLatency caps a config-declared latency/jittered window. Config is an
// untrusted trust boundary (hot-reloaded files, HTTP admin), so an absurd sleep
// must be rejected. not silently honored.
const maxFaultLatency = 5 * time.Minute

func buildFault(fs FaultSpec) (fault.Fault, error) {
	switch fs.Type {
	case "latency":
		d, err := time.ParseDuration(fs.Duration)
		if err != nil {
			return nil, fmt.Errorf("chaotic: latency duration %q: %w", fs.Duration, err)
		}
		if d < 0 || d > maxFaultLatency {
			return nil, fmt.Errorf("chaotic: latency %s out of (0, %s]", d, maxFaultLatency)
		}
		return fault.Latency(d), nil
	case "jittered":
		minD, err := time.ParseDuration(fs.Min)
		if err != nil {
			return nil, fmt.Errorf("chaotic: jittered min %q: %w", fs.Min, err)
		}
		maxD, err := time.ParseDuration(fs.Max)
		if err != nil {
			return nil, fmt.Errorf("chaotic: jittered max %q: %w", fs.Max, err)
		}
		if minD < 0 || maxD > maxFaultLatency {
			return nil, fmt.Errorf("chaotic: jittered window [%s,%s] out of (0, %s]", minD, maxD, maxFaultLatency)
		}
		return fault.Jittered(minD, maxD), nil
	case "error":
		msg := fs.Message
		if msg == "" {
			msg = "chaotic: injected error"
		}
		return fault.Error(errors.New(msg)), nil
	case "panic":
		return fault.Panic(fs.Value), nil
	case "conn_drop":
		return fault.ConnDrop(), nil
	}
	return nil, fmt.Errorf("chaotic: unknown fault type %q", fs.Type)
}
