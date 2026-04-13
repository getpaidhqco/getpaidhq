package lib

import (
	"log/slog"
	"os"
)

var globalLogger Logger

// GetLogger returns the global logger instance, creating it if needed.
func GetLogger() Logger {
	if globalLogger == nil {
		globalLogger = newLogger(NewEnv())
	}
	return globalLogger
}

// GinLogger wraps Logger for gin-framework's io.Writer interface.
type GinLogger struct {
	Logger
}

// Write implements io.Writer for gin-framework logging.
func (l GinLogger) Write(p []byte) (n int, err error) {
	l.Info(string(p))
	return len(p), nil
}

type slogLogger struct {
	logger *slog.Logger
}

func (l slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l slogLogger) Fatal(msg string, args ...any) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}

func (l slogLogger) Sync() error {
	return nil
}

// GetSlogLogger returns the underlying *slog.Logger for adapters that need it.
func GetSlogLogger() *slog.Logger {
	if globalLogger == nil {
		GetLogger()
	}
	if sl, ok := globalLogger.(slogLogger); ok {
		return sl.logger
	}
	return slog.Default()
}

func newLogger(env Env) Logger {
	var level slog.Level
	switch env.LogLevel {
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: env.Env == "development",
	}

	var handler slog.Handler
	if env.Env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	l := slog.New(handler)
	slog.SetDefault(l)

	return slogLogger{logger: l}
}
