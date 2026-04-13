package lib

// Logger is the core logging interface used throughout the application.
// All methods use slog-style structured key-value pairs: logger.Info("msg", "key", val, "key2", val2)
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Fatal(msg string, args ...any)

	Sync() error
}
