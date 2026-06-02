package file_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/source/file"
)

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func TestWatchLoadsInitialThenReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	writeFile(t, path, "rules:\n  - name: a\n    kinds: [http_client]\n    faults: [{type: error, message: x}]\n")

	eng := engine.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	go func() { _ = file.Watch(ctx, path, eng, logger) }()

	waitFor(t, 2*time.Second, func() bool {
		return eng.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient}) != engine.Pass
	})

	writeFile(t, path, "rules:\n  - name: b\n    kinds: [sql]\n    faults: [{type: error, message: y}]\n")
	waitFor(t, 3*time.Second, func() bool {
		sqlFires := eng.Eval(context.Background(), engine.Op{Kind: engine.OpSQL}) != engine.Pass
		httpStopped := eng.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient}) == engine.Pass
		return sqlFires && httpStopped
	})
}
