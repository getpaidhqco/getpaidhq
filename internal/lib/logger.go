package lib

import (
	"fmt"
	"log"
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
	l.Infof(string(p))
	return len(p), nil
}

type slogLogger struct {
	logger *slog.Logger
}

func (l slogLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(msg, keysAndValues...)
}

func (l slogLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info(msg, keysAndValues...)
}

func (l slogLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Warn(msg, keysAndValues...)
}

func (l slogLogger) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Error(msg, keysAndValues...)
}

func (l slogLogger) Sync() error {
	return nil
}

func (l slogLogger) Fatalf(msg string, keysAndValues ...interface{}) {
	log.Fatal(msg, keysAndValues)
}

func (l slogLogger) Fatal(msg string, keysAndValues ...interface{}) {
	log.Fatal(msg, keysAndValues)
}

func (l slogLogger) Infof(template string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(template, args...))
}

func (l slogLogger) Debugf(template string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(template, args...))
}

func (l slogLogger) Errorf(template string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(template, args...))
}

func (l slogLogger) Panicf(template string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(template, args...))
}

func (l slogLogger) Warnf(template string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(template, args...))
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
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
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
