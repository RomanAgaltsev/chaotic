package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/RomanAgaltsev/chaotic/fault"
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
	Stages   []StageSpec       `yaml:"stages" json:"stages"`
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
	Rate     int    `yaml:"rate" json:"rate"`         // slow_reader / slow_writer (bytes/sec)
	Limit    int    `yaml:"limit" json:"limit"`       // truncate (bytes)
}

// StageSpec is the serializable form of a Stage. Times == 0 in the final stage
// means "forever".
type StageSpec struct {
	Times  int         `yaml:"times" json:"times"`
	Faults []FaultSpec `yaml:"faults" json:"faults"`
}

var kindNames = map[string]Kind{
	"http_client": OpHTTPClient,
	"http_server": OpHTTPServer,
	"sql":         OpSQL,
	"grpc_client": OpGRPCClient,
	"grpc_server": OpGRPCServer,
	"explicit":    OpExplicit,
	"pgx":         OpPGX,
	"redis":       OpRedis,
	"rabbitmq":    OpRabbitMQ,
	"mongo":       OpMongo,
	"kafka":       OpKafka,
	"aws":         OpAWS,
	"nats":        OpNATS,
	"net":         OpNet,
	"io":          OpIO,
}

// BuildRule converts a RuleSpec into a Rule, validating kinds, counter type,
// fault types, durations, and probability bounds. It is total: it never panics,
// and it aggregates every invalid field into a single errors.Join result so a
// caller (and the operator editing config) sees all problems at once.
func BuildRule(spec RuleSpec) (Rule, error) {
	var opts []RuleOption
	var errs []error

	mopts, merrs := buildMatchOptions(spec)
	opts = append(opts, mopts...)
	errs = append(errs, merrs...)

	if copt, err := buildCounterOption(spec.Counter); err != nil {
		errs = append(errs, err)
	} else {
		opts = append(opts, copt)
	}

	faults, ferrs := buildFaultList(spec.Faults)
	errs = append(errs, ferrs...)
	if len(faults) > 0 {
		opts = append(opts, WithFaults(faults...))
	}

	if len(spec.Stages) > 0 {
		stages, serrs := buildStageOptions(spec)
		errs = append(errs, serrs...)
		if len(errs) == 0 {
			opts = append(opts, WithStages(stages...))
		}
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

// buildMatchOptions turns the spec's match fields (kinds, name glob, attrs) into
// RuleOptions, collecting one error per unknown kind.
func buildMatchOptions(spec RuleSpec) ([]RuleOption, []error) {
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
	return opts, errs
}

// buildCounterOption maps a CounterSpec to its RuleOption, returning an error for
// an out-of-range probability or an unknown counter type.
func buildCounterOption(c CounterSpec) (RuleOption, error) {
	switch c.Type {
	case "", "always":
		return Always(), nil
	case "times":
		return Times(c.N), nil
	case "range":
		return Range(c.From, c.To), nil
	case "probability":
		if c.P < 0 || c.P > 1 {
			return nil, fmt.Errorf("chaotic: probability p=%v out of [0,1]", c.P)
		}
		return Probability(c.P, c.Seed), nil
	default:
		return nil, fmt.Errorf("chaotic: unknown counter type %q", c.Type)
	}
}

// buildFaultList builds every fault in specs, collecting one error per invalid
// entry and skipping it.
func buildFaultList(specs []FaultSpec) ([]fault.Fault, []error) {
	faults := make([]fault.Fault, 0, len(specs))
	var errs []error
	for _, fs := range specs {
		f, err := buildFault(fs)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		faults = append(faults, f)
	}
	return faults, errs
}

// buildStageOptions validates the staged-schedule constraints and builds each
// stage's faults, aggregating every problem into errs.
func buildStageOptions(spec RuleSpec) ([]Stage, []error) {
	var errs []error
	if spec.Counter.Type != "" && spec.Counter.Type != "always" {
		errs = append(errs, errors.New("chaotic: stages cannot be combined with a counter"))
	}
	if len(spec.Faults) > 0 {
		errs = append(errs, errors.New("chaotic: stages cannot be combined with top-level faults"))
	}
	stages := make([]Stage, 0, len(spec.Stages))
	for i, ss := range spec.Stages {
		last := i == len(spec.Stages)-1
		if !last && ss.Times <= 0 {
			errs = append(errs, fmt.Errorf("chaotic: stage %d: only the final stage may have Times <= 0", i))
		}
		if last && ss.Times < 0 {
			errs = append(errs, errors.New("chaotic: final stage Times must be >= 0"))
		}
		sf, ferrs := buildFaultList(ss.Faults)
		errs = append(errs, ferrs...)
		stages = append(stages, Stage{Times: ss.Times, Faults: sf})
	}
	return stages, errs
}

// maxFaultLatency caps a config-declared latency/jittered window. Config is an
// untrusted trust boundary (hot-reloaded files, HTTP admin), so an absurd sleep
// must be rejected. not silently honored.
const maxFaultLatency = 5 * time.Minute

func buildFault(fs FaultSpec) (fault.Fault, error) {
	switch fs.Type {
	case "latency":
		return buildLatencyFault(fs)
	case "jittered":
		return buildJitteredFault(fs)
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
	case "disconnect":
		return fault.Disconnect(), nil
	case "slow_reader":
		return buildRateFault("slow_reader", fs.Rate, fault.SlowReader)
	case "slow_writer":
		return buildRateFault("slow_writer", fs.Rate, fault.SlowWriter)
	case "truncate":
		if fs.Limit < 0 {
			return nil, fmt.Errorf("chaotic: truncate limit %d must be >= 0", fs.Limit)
		}
		return fault.Truncate(fs.Limit), nil
	}
	return nil, fmt.Errorf("chaotic: unknown fault type %q", fs.Type)
}

// buildLatencyFault parses and bounds-checks a "latency" fault's duration.
func buildLatencyFault(fs FaultSpec) (fault.Fault, error) {
	d, err := time.ParseDuration(fs.Duration)
	if err != nil {
		return nil, fmt.Errorf("chaotic: latency duration %q: %w", fs.Duration, err)
	}
	if d < 0 || d > maxFaultLatency {
		return nil, fmt.Errorf("chaotic: latency %s out of (0, %s]", d, maxFaultLatency)
	}
	return fault.Latency(d), nil
}

// buildJitteredFault parses and bounds-checks a "jittered" fault's window.
func buildJitteredFault(fs FaultSpec) (fault.Fault, error) {
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
}

// buildRateFault validates a non-negative byte rate and constructs the fault via
// mk. Shared by slow_reader and slow_writer.
func buildRateFault(name string, rate int, mk func(int) fault.Fault) (fault.Fault, error) {
	if rate < 0 {
		return nil, fmt.Errorf("chaotic: %s rate %d must be >= 0", name, rate)
	}
	return mk(rate), nil
}
