//go:build !chaos_off

package redis

import (
	"context"
	"errors"
	"io"
	"net"

	goredis "github.com/redis/go-redis/v9"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// firstKey returns a best-effort key for an Op's Attrs. go-redis stores the
// command verb in Args()[0]. The first key, when present, is Args()[1]. It is
// advisory match metadata, not correctness-bearing: commands whose second arg
// is not a string key yield "".
func firstKey(cmd goredis.Cmder) string {
	args := cmd.Args()
	if len(args) < 2 {
		return ""
	}
	if s, ok := args[1].(string); ok {
		return s
	}
	return ""
}

// NewHook returns a redis.Hook that consults eng on every command, pipeline
// and dial. When eng is disabled or has no rules, every path is a newar-zero-cost
// passthrough.
func NewHook(eng *engine.Engine) goredis.Hook {
	return chaosHook{eng: eng}
}

type chaosHook struct {
	eng *engine.Engine
}

func (h chaosHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		if !h.eng.Enabled() {
			return next(ctx, cmd)
		}
		op := engine.Op{
			Kind:   engine.OpRedis,
			Name:   cmd.Name(),
			Method: "single",
			Attrs:  map[string]string{"key": firstKey(cmd)},
		}
		action := h.eng.Eval(ctx, op)
		if err := action.Before(ctx); err != nil {
			reportOutcome(ctx, action, err)
			mapped := mapErr(err)
			cmd.SetErr(mapped)
			return mapped
		}
		err := next(ctx, cmd)
		reportOutcome(ctx, action, err)
		return err
	}
}

func (h chaosHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !h.eng.Enabled() {
			return next(ctx, network, addr)
		}
		op := engine.Op{
			Kind:   engine.OpRedis,
			Name:   "DIAL",
			Method: "dial",
		}
		action := h.eng.Eval(ctx, op)
		if err := action.Before(ctx); err != nil {
			reportOutcome(ctx, action, err)
			return nil, mapErr(err)
		}
		conn, err := next(ctx, network, addr)
		reportOutcome(ctx, action, err)
		return conn, err
	}
}

func (h chaosHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		if !h.eng.Enabled() || len(cmds) == 0 {
			return next(ctx, cmds)
		}
		op := engine.Op{
			Kind:   engine.OpRedis,
			Name:   "pipeline",
			Method: pipelineMethod(cmds),
		}
		action := h.eng.Eval(ctx, op)
		if err := action.Before(ctx); err != nil {
			reportOutcome(ctx, action, err)
			mapped := mapErr(err)
			for _, c := range cmds {
				c.SetErr(mapped)
			}
			return mapped
		}
		err := next(ctx, cmds)
		reportOutcome(ctx, action, err)
		return err
	}
}

// pipelineMethod reports "tx" when the batch is a MULTI/EXEC transaction
// (go-redis wraps TxPipeline batches with a leading MULTI), else "pipeline".
func pipelineMethod(cmds []goredis.Cmder) string {
	if len(cmds) > 0 && cmds[0].Name() == "multi" {
		return "tx"
	}
	return "pipeline"
}

// mapErr translates a fault error into go-redis's native error model. ConnDrop
// becomes the broken-pipe shape go-redis sees on real dead connection, which
// engages its pool eviction and retry logic. Every other fault error passes
// through unchanged (callers set whatever error they want via fault.Error).
func mapErr(err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return &net.OpError{
			Op:  "read",
			Net: "tcp",
			Err: io.ErrUnexpectedEOF,
		}
	}
	return err
}

// reportOutcome forwards the call's error (or the injected fault) to the engine
// when the action reports outcomes, then runs After to release any held bound
// (e.g. a WithMaxConcurrent slot). Call it exactly once per action, or the slot
// leaks and the failure budget never sees the call. A nil action is a no-op.
func reportOutcome(ctx context.Context, action engine.Action, callErr error) {
	if action == nil {
		return
	}
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
	_ = action.After(ctx)
}
