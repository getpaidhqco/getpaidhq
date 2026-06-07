package lib

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	globalLogger Logger
	slogLogger   *slog.Logger
	loggerOnce   sync.Once
)

// GetLogger returns the global logger instance, creating it once on first call.
// The sync.Once makes it safe to call from concurrent goroutines (pubsub
// handlers, workers, HTTP handlers all reach for it); without it, racing first
// callers ran newLogger(NewEnv()) — and viper's map state — in parallel.
func GetLogger() Logger {
	loggerOnce.Do(func() {
		globalLogger = newLogger(NewEnv())
	})
	return globalLogger
}

// GetSlogLogger returns the underlying *slog.Logger so adapters that need a
// raw slog instance (e.g. the Temporal SDK structured-logger adapter) can use
// it. Triggers logger initialization on first call.
func GetSlogLogger() *slog.Logger {
	GetLogger() // ensures slogLogger is initialized via the once
	return slogLogger
}

type MyLogger struct {
	logger *slog.Logger
}

// emit writes a record at the given level, attributing the source to the
// real caller instead of this wrapper.
//
// slog's AddSource walks a fixed number of stack frames from wherever the
// slog.Logger.Info/Warn/... call happens. Because every call here funnels
// through a MyLogger method, that frame is always this file — which is why
// every log line used to report `internal/lib/logger.go`. We capture the
// caller's PC ourselves and hand slog a Record built with it, so the source
// points at the code that actually logged. `skip` is measured from emit's
// caller (the exported MyLogger method): see the per-method callers below.
func (l MyLogger) emit(skip int, level slog.Level, msg string, attrs ...any) {
	ctx := context.Background()
	if !l.logger.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	// skip frames: runtime.Callers, this emit, then `skip` more to reach the
	// original caller (1 for a direct method, 2 when a *f helper calls emit).
	runtime.Callers(2+skip, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(attrs...)
	_ = l.logger.Handler().Handle(ctx, r)
}

func (l MyLogger) Debug(msg string, keysAndValues ...any) {
	l.emit(1, slog.LevelDebug, msg, keysAndValues...)
}

func (l MyLogger) Info(msg string, keysAndValues ...any) {
	l.emit(1, slog.LevelInfo, msg, keysAndValues...)
}

func (l MyLogger) Warn(msg string, keysAndValues ...any) {
	l.emit(1, slog.LevelWarn, msg, keysAndValues...)
}

func (l MyLogger) Error(msg string, keysAndValues ...any) {
	l.emit(1, slog.LevelError, msg, keysAndValues...)
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
	l.emit(1, slog.LevelInfo, fmt.Sprintf(template, args...))
}
func (l MyLogger) Debugf(template string, args ...any) {
	l.emit(1, slog.LevelDebug, fmt.Sprintf(template, args...))
}
func (l MyLogger) Errorf(template string, args ...any) {
	l.emit(1, slog.LevelError, fmt.Sprintf(template, args...))
}
func (l MyLogger) Panicf(template string, args ...any) {
	// Previously this only logged at error level, contradicting the method
	// name and silently passing through paths that meant to halt. Callers
	// must be able to rely on Panicf actually panicking.
	msg := fmt.Sprintf(template, args...)
	l.emit(1, slog.LevelError, msg)
	panic(msg)
}
func (l MyLogger) Warnf(template string, args ...any) {
	l.emit(1, slog.LevelWarn, fmt.Sprintf(template, args...))
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
	// Make this the process-wide default so libraries that log through the
	// default slog logger (Fuego's "JSON spec:" / "OpenAPI UI:" startup
	// messages, and anything else using slog.Info directly) share our format
	// and level instead of falling back to the stdlib log-style default.
	slog.SetDefault(slogLogger)
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
