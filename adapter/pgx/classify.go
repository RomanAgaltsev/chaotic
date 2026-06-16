package pgx

import (
	"strconv"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/internal/sqlclass"
)

// boolStr returns "true" or "false".
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// opQuery builds the Op for Query, QueryRow, or Exec on a wrapped pool/conn/tx.
// method is "query", "queryrow", or "exec". inTx is true when the caller is a *Tx.
func opQuery(method, sql string, nArgs int, inTx bool) engine.Op {
	c := sqlclass.Classify(sql)
	name := c.Verb
	if c.Table != "" {
		name = c.Verb + " " + c.Table
	}
	return engine.Op{
		Kind:   engine.OpPGX,
		Name:   name,
		Method: method,
		Attrs: map[string]string{
			"table": c.Table,
			"args":  strconv.Itoa(nArgs),
			"tx":    boolStr(inTx),
			"sql":   sql,
		},
	}
}

// opBatch builds the Op for SendBatch. size is the batch's queued statement count.
func opBatch(size int, inTx bool) engine.Op {
	return engine.Op{
		Kind:   engine.OpPGX,
		Name:   "BATCH",
		Method: "batch",
		Attrs: map[string]string{
			"batch_size": strconv.Itoa(size),
			"tx":         boolStr(inTx),
		},
	}
}

// opBegin builds the Op for Begin/BeginTx. iso/access/deferrable carry the
// resolved TxOptions; pass empty strings / false for plain Begin().
func opBegin(iso, access string, deferrable bool) engine.Op {
	return engine.Op{
		Kind:   engine.OpPGX,
		Name:   "BEGIN",
		Method: "begin",
		Attrs: map[string]string{
			"iso_level":   iso,
			"access_mode": access,
			"deferrable":  boolStr(deferrable),
		},
	}
}

// opAcquire builds the Op for Pool.Acquire.
func opAcquire() engine.Op {
	return engine.Op{
		Kind:   engine.OpPGX,
		Name:   "ACQUIRE",
		Method: "acquire",
		Attrs:  nil,
	}
}

// opTrace builds the Op for the config-level QueryTracer path. The tracer fires
// for Query, QueryRow and Exec indistinguishably, so Method is the fixed label
// "trace" (rule authors can match or exclude the config path with it). Name and
// the table attr come from the same SQL classifier the direct path uses.
func opTrace(sql string, nArgs int) engine.Op {
	c := sqlclass.Classify(sql)
	name := c.Verb
	if c.Table != "" {
		name = c.Verb + " " + c.Table
	}
	return engine.Op{
		Kind:   engine.OpPGX,
		Name:   name,
		Method: "trace",
		Attrs: map[string]string{
			"table": c.Table,
			"args":  strconv.Itoa(nArgs),
			"sql":   sql,
		},
	}
}
