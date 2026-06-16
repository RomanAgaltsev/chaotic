package file

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/fsnotify/fsnotify"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// Watch loads path into eng once, then watches it and calls eng.ReplaceRules on
// every successful reload. A parse error keeps the previous rules and is logged
// via logger (slog.Default if nil). Watch blocks until ctx is canceled.
func Watch(ctx context.Context, path string, eng *engine.Engine, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	if _, err := reload(path, eng); err != nil {
		return err
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	if err := w.Add(dir); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			handleFileEvent(ev, path, base, eng, logger)
		case werr, ok := <-w.Errors:
			if !ok {
				return nil
			}
			logger.Warn("chaotic: watcher error", "error", werr)
		}
	}
}

// handleFileEvent reloads eng when ev is a content change (write or create) to
// the watched file. Events on other files, non-content ops, and parse failures
// are ignored (the latter logged, keeping the previous rules).
func handleFileEvent(ev fsnotify.Event, path, base string, eng *engine.Engine, logger *slog.Logger) {
	if filepath.Base(ev.Name) != base {
		return
	}
	if ev.Op&(fsnotify.Write|fsnotify.Create) == 0 {
		return
	}
	n, lerr := reload(path, eng)
	if lerr != nil {
		logger.Warn("chaotic: rule reload failed, keeping previous rules", "error", lerr)
		return
	}
	logger.Info("chaotic: rule reloaded", "count", n)
}

// reload loads path and atomically swaps eng's rules. It recovers any panic
// from the load/build path and converts it to an error, so a watcher goroutine
// can never be killed by malformed rules file.
func reload(path string, eng *engine.Engine) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("chaotic: panic during rule reload: %v", r)
		}
	}()
	rs, lerr := Load(path)
	if lerr != nil {
		return 0, lerr
	}
	eng.ReplaceRules(rs)
	return rs.Len(), nil
}
