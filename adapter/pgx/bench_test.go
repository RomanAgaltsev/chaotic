package pgx

import (
	"context"
	"testing"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func benchPool(eng *engine.Engine) *Pool {
	return newPoolWithFake(eng, &fakePool{
		onExec:      func(context.Context, string, ...any) (pgconn.CommandTag, error) { return pgconn.CommandTag{}, nil },
		onQuery:     func(context.Context, string, ...any) (pgxv5.Rows, error) { return nil, nil },
		onQueryRow:  func(context.Context, string, ...any) pgxv5.Row { return chaosRow{} },
		onSendBatch: func(context.Context, *pgxv5.Batch) pgxv5.BatchResults { return chaosBatch{} },
	})
}

func BenchmarkPoolExecNoRules(b *testing.B) {
	p := benchPool(engine.New())
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = p.Exec(ctx, "INSERT INTO t VALUES (1)")
	}
}

func BenchmarkPoolQueryNoRules(b *testing.B) {
	p := benchPool(engine.New())
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = p.Query(ctx, "SELECT 1")
	}
}

func BenchmarkPoolExecRuleNoMatch(b *testing.B) {
	// Rule installed but matches nothing (different Kind).
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(fault.Error(nil)),
	))
	p := benchPool(eng)
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = p.Exec(ctx, "INSERT INTO t VALUES (1)")
	}
}

func BenchmarkPoolExecRuleMatchPass(b *testing.B) {
	// Rule matches but is "exhausted" — Times(0) effectively means no-op.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.Times(0),
		engine.WithFault(fault.Error(nil)),
	))
	p := benchPool(eng)
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = p.Exec(ctx, "INSERT INTO t VALUES (1)")
	}
}
