package main

import (
	"context"
	"log/slog"
	"time"

    "github.com/ssvlabs/go-ssv-scanner/internal/eth"
    appLogger "github.com/ssvlabs/go-ssv-scanner/internal/logger"
)

// startProgressTicker starts a central Info-level progress ticker for a scanner
func startProgressTicker(ctx context.Context, l *slog.Logger, prog *eth.ScanProgress) appLogger.StopFunc {
	return appLogger.StartTicker(ctx, l, 2*time.Second, func(lg *slog.Logger) {
		name, from, to, latest, _, window, total, done := prog.Snapshot()
		var pct any
		if total > 0 {
			// Use actual cumulative progress when available
			p := float64(done) / float64(total) * 100
			if p < 0 {
				p = 0
			}
			if p > 100 {
				p = 100
			}
			pct = int(p + 0.5)
		} else {
			pct = nil
		}
		// Include total/done for clarity when present
		if pct != nil {
			lg.Info("scan progress", "scanner", name, "from", from, "to", to, "latest", latest, "window", window, "done", done, "total", total, "%", pct)
		} else {
			lg.Info("scan progress", "scanner", name, "from", from, "to", to, "latest", latest, "window", window)
		}
	})
}
