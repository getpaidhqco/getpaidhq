package lib

import (
	"fmt"
	"log"
	"log/slog"
	"os"
)

var (
	globalLogger Logger
	slogLogger   *slog.Logger
)

// GetLogger returns the global logger instance, creating it if needed.
func GetLogger() Logger {
	if globalLogger == nil {
		globalLogger = newLogger(NewEnv())
	}
	return globalLogger
}

// GetSlogLogger returns the underlying *slog.Logger so adapters that need a
// raw slog instance (e.g. the Temporal SDK structured-logger adapter) can use
// it. Triggers logger initialization on first call.
func GetSlogLogger() *slog.Logger {
	if slogLogger == nil {
		_ = GetLogger()
	}
	return slogLogger
}

type MyLogger struct {
	logger *slog.Logger
}

func (l MyLogger) Debug(msg string, keysAndValues ...any) {
	l.logger.Debug(msg, keysAndValues...)
}

func (l MyLogger) Info(msg string, keysAndValues ...any) {
	l.logger.Info(msg, keysAndValues...)
}

func (l MyLogger) Warn(msg string, keysAndValues ...any) {
	l.logger.Warn(msg, keysAndValues...)
}

func (l MyLogger) Error(msg string, keysAndValues ...any) {
	l.logger.Error(msg, keysAndValues...)
}

func (l MyLogger) Sync() error {
	return nil
}

func (l MyLogger) Fatalf(msg string, keysAndValues ...any) {
	log.Fatal(msg, keysAndValues)
}

func (l MyLogger) Fatal(msg string, keysAndValues ...any) {
	log.Fatal(msg, keysAndValues)
}

func (l MyLogger) Infof(template string, args ...any) {
	l.logger.Info(fmt.Sprintf(template, args...))
}
func (l MyLogger) Debugf(template string, args ...any) {
	l.logger.Debug(fmt.Sprintf(template, args...))
}
func (l MyLogger) Errorf(template string, args ...any) {
	l.logger.Error(fmt.Sprintf(template, args...))
}
func (l MyLogger) Panicf(template string, args ...any) {
	l.logger.Error(fmt.Sprintf(template, args...))
}
func (l MyLogger) Warnf(template string, args ...any) {
	l.logger.Warn(fmt.Sprintf(template, args...))
}

// newLogger sets up the structured logger backed by log/slog.
func newLogger(env Env) Logger {
	level := parseLogLevel(env.LogLevel)
	output := resolveLogOutput(env)

	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	var handler slog.Handler
	if env.Env == "development" {
		handler = slog.NewTextHandler(output, withoutTime(handlerOpts))
	} else {
		handler = slog.NewJSONHandler(output, handlerOpts)
	}

	slogLogger = slog.New(handler)
	return MyLogger{logger: slogLogger}
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "fatal":
		// slog has no Fatal level; map to Error. Fatal/Fatalf on the Logger
		// interface still terminates via the stdlib log package.
		return slog.LevelError
	default:
		// Unknown level → silence everything below panic.
		return slog.LevelError + 4
	}
}

func resolveLogOutput(env Env) *os.File {
	if env.Env == "production" && env.LogOutput != "" {
		if f, err := os.OpenFile(env.LogOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			return f
		}
	}
	return os.Stderr
}

func withoutTime(opts *slog.HandlerOptions) *slog.HandlerOptions {
	clone := *opts
	prev := clone.ReplaceAttr
	clone.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		if len(groups) == 0 && a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		if prev != nil {
			return prev(groups, a)
		}
		return a
	}
	return &clone
}
