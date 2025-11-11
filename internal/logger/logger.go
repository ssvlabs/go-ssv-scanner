package logger

import (
    "fmt"
    "io"
    "log/slog"
    "os"
    "path/filepath"
    "time"
)

// GlobalInit initializes a slog logger that writes JSON logs to stderr and a timestamped file.
// It also sets the logger as slog's default for easy use across packages.
func GlobalInit() (*slog.Logger, error) {
    ts := GetRunTimestamp()
    dir := BuildLogDir(ts)
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return nil, err
    }

    f, err := os.OpenFile(TimestampedLogPath(ts), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil {
        return nil, err
    }

    // Use stderr to avoid polluting stdout (kept for JSON command outputs)
    mw := io.MultiWriter(os.Stderr, f)
    handler := slog.NewJSONHandler(mw, &slog.HandlerOptions{Level: slog.LevelInfo})
    l := slog.New(handler)
    slog.SetDefault(l)
    return l, nil
}

// SetLogLevel changes the default logger level dynamically.
func SetLogLevel(l *slog.Logger, level slog.Level) {
    // recreate handler with new level; keep writing to stderr only for dynamic change
    h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
    *l = *slog.New(h)
    l.Info("log level changed", "level", level.String())
}

// GetRunTimestamp returns a compact timestamp for folder/file naming.
func GetRunTimestamp() string { return time.Now().Format("20060102-150405") }

// BuildLogDir returns the directory path for a run.
func BuildLogDir(ts string) string { return filepath.Join("logs", ts) }

// TimestampedLogPath returns the JSON log file path for a run.
func TimestampedLogPath(ts string) string { return filepath.Join(BuildLogDir(ts), fmt.Sprintf("%s.log", ts)) }

