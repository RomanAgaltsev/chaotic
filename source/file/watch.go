package file

import (
	"context"
	"log/slog"

	"github.com/ag4r/chaotic/engine"
	"github.com/fsnotify/fsnotify"
)

// Watch loads path into eng once, then watches it and calls eng.ReplaceRules on
// every successful reload. A parse error keeps the previous rules and is logged
// via logger (slog.Default if nil). Watch blocks until ctx is canceled.
func Watch(ctx context.Context, path string, eng *engine.Engine, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	rs, err := Load(path)
	if err != nil {
		return err
	}
	eng.ReplaceRules(rs)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()
	if err := w.Add(path); err != nil {
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
			if ev.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			newRS, lerr := Load(path)
			if lerr != nil {
				logger.Warn("chaotic: rule reload failed, keeping previous rules", "error", lerr)
				continue
			}
			eng.ReplaceRules(newRS)
			logger.Info("chaotic: rule reloaded", "count", newRS.Len())
		case werr, ok := <-w.Errors:
			if !ok {
				return nil
			}
			logger.Warn("chaotic: watcher error", "error", werr)
		}
	}
}
