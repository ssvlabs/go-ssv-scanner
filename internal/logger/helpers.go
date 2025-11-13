package logger

import (
	"context"
	"log/slog"
	"time"
)

// StopFunc is a function that stops a periodic logger
type StopFunc func()

// StartTicker starts a periodic logger that invokes logFn at the given interval; stops on ctx.Done or returned Stop()
func StartTicker(ctx context.Context, l *slog.Logger, interval time.Duration, logFn func(l *slog.Logger)) StopFunc {
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				logFn(l)
			case <-done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return func() { close(done) }
}
