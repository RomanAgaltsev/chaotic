package pgx

import (
	"testing"

	"github.com/ag4r/chaotic/engine"
)

func TestOpQueryBuildsExpectedOp(t *testing.T) {
	op := opQuery("query", "SELECT * FROM users WHERE id = $1", 1, false)
	if op.Kind != engine.OpPGX {
		t.Errorf("Kind = %v, want OpPGX", op.Kind)
	}
	if op.Name != "SELECT users" {
		t.Errorf("Name = %q, want %q", op.Name, "SELECT users")
	}
	if op.Method != "query" {
		t.Errorf("Method = %q, want %q", op.Method, "query")
	}
	if op.Attrs["table"] != "users" {
		t.Errorf("Attrs[table] = %q, want %q", op.Attrs["table"], "users")
	}
	if op.Attrs["args"] != "1" {
		t.Errorf("Attrs[args] = %q, want %q", op.Attrs["args"], "1")
	}
	if op.Attrs["tx"] != "false" {
		t.Errorf("Attrs[tx] = %q, want %q", op.Attrs["tx"], "false")
	}
	if op.Attrs["sql"] != "SELECT * FROM users WHERE id = $1" {
		t.Errorf("Attrs[sql] = %q, want raw query", op.Attrs["sql"])
	}
}

func TestOpQueryUnclassifiable(t *testing.T) {
	op := opQuery("exec", "", 0, true)
	if op.Name != "" {
		t.Errorf("Name = %q, want empty (no verb)", op.Name)
	}
	if op.Attrs["tx"] != "true" {
		t.Errorf("Attrs[tx] = %q, want true", op.Attrs["tx"])
	}
}

func TestOpBatch(t *testing.T) {
	op := opBatch(5, true)
	if op.Name != "BATCH" {
		t.Errorf("Name = %q, want BATCH", op.Name)
	}
	if op.Method != "batch" {
		t.Errorf("Method = %q, want batch", op.Method)
	}
	if op.Attrs["batch_size"] != "5" {
		t.Errorf("Attrs[batch_size] = %q, want 5", op.Attrs["batch_size"])
	}
	if op.Attrs["tx"] != "true" {
		t.Errorf("Attrs[tx] = %q, want true", op.Attrs["tx"])
	}
}

func TestOpBegin(t *testing.T) {
	op := opBegin("read committed", "read only", true)
	if op.Name != "BEGIN" || op.Method != "begin" {
		t.Errorf("Name/Method = %q/%q, want BEGIN/begin", op.Name, op.Method)
	}
	if op.Attrs["iso_level"] != "read committed" {
		t.Errorf("Attrs[iso_level] = %q", op.Attrs["iso_level"])
	}
	if op.Attrs["deferrable"] != "true" {
		t.Errorf("Attrs[deferrable] = %q", op.Attrs["deferrable"])
	}
}

func TestOpAcquire(t *testing.T) {
	op := opAcquire()
	if op.Name != "ACQUIRE" || op.Method != "acquire" {
		t.Errorf("Name/Method = %q/%q, want ACQUIRE/acquire", op.Name, op.Method)
	}
	if len(op.Attrs) != 0 {
		t.Errorf("Attrs = %v, want empty", op.Attrs)
	}
}
