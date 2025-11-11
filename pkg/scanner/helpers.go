package scanner

import (
	"context"
	"log/slog"
	"time"

    "github.com/ssvlabs/go-ssv-scanner/internal/eth"
    appLogger "github.com/ssvlabs/go-ssv-scanner/internal/logger"
)

// maybeProgress sets up a periodic info-level progress log to the provided logger
// if enabled via config. Returns the ScanProgress and a stop function.
func (s *Scanner) maybeProgress(ctx context.Context, l *slog.Logger, name string) (*eth.ScanProgress, appLogger.StopFunc) {
	if !s.cfg.EnableProgress || l == nil {
		return nil, nil
	}
	prog := &eth.ScanProgress{}
	prog.SetName(name)
	interval := s.cfg.ProgressInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	stop := appLogger.StartTicker(ctx, l, interval, func(lg *slog.Logger) {
		name, from, to, latest, _, window, total, done := prog.Snapshot()
		var pct any
		if total > 0 {
			p := float64(done) / float64(total) * 100
			if p < 0 {
				p = 0
			}
			if p > 100 {
				p = 100
			}
			pct = int(p + 0.5)
		}
		if pct != nil {
			lg.Info("scan progress", "scanner", name, "from", from, "to", to, "latest", latest, "window", window, "done", done, "total", total, "%", pct)
		} else {
			lg.Info("scan progress", "scanner", name, "from", from, "to", to, "latest", latest, "window", window)
		}
	})
	return prog, stop
}
