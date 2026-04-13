package lib

import (
	"fmt"
	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"log/slog"
)

var (
	globalLogger Logger
	zapLogger    *zap.Logger
)

// GinLogger wraps Logger for gin-framework's io.Writer interface.
type GinLogger struct {
	Logger
}

// GetLogger returns the global logger instance, creating it if needed.
func GetLogger() Logger {
	if globalLogger == nil {
		ll := newLogger(NewEnv(), zap.WithCaller(true))
		globalLogger = ll
	}
	return globalLogger
}

type MyLogger struct {
	logger *slog.Logger
}

func (l MyLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(msg, keysAndValues...)
}

func (l MyLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info(msg, keysAndValues...)
}

func (l MyLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Warn(msg, keysAndValues...)
}

func (l MyLogger) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Error(msg, keysAndValues...)
}

func (l MyLogger) Sync() error {
	return nil
}

func (l MyLogger) Fatalf(msg string, keysAndValues ...interface{}) {
	log.Fatal(msg, keysAndValues)
}

func (l MyLogger) Fatal(msg string, keysAndValues ...interface{}) {
	log.Fatal(msg, keysAndValues)
}

func (l MyLogger) Infof(template string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(template, args...))
}
func (l MyLogger) Debugf(template string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(template, args...))
}
func (l MyLogger) Errorf(template string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(template, args...))
}
func (l MyLogger) Panicf(template string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(template, args...))
}
func (l MyLogger) Warnf(template string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(template, args...))
}

// newLogger sets up the structured logger backed by zap.
func newLogger(env Env, opts ...zap.Option) Logger {
	config := zap.NewDevelopmentConfig()
	logOutput := env.LogOutput

	if env.Env == "development" {
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	if env.Env == "production" && logOutput != "" {
		config.OutputPaths = []string{logOutput}
	}

	logLevel := env.LogLevel
	level := zap.PanicLevel
	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "fatal":
		level = zapcore.FatalLevel
	default:
		level = zap.PanicLevel
	}
	opts = append(opts, zap.AddStacktrace(zapcore.FatalLevel))
	config.Level.SetLevel(level)

	if env.Env != "development" {
		config.EncoderConfig.TimeKey = ""
	}

	zapLogger, _ = config.Build(opts...)
	handler := slogzap.Option{
		Level:     slog.LevelDebug,
		Logger:    zapLogger,
		AddSource: false,
	}.NewZapHandler()

	l := slog.New(handler)
	return MyLogger{
		logger: l,
	}
}

// Write implements io.Writer for gin-framework logging.
func (l GinLogger) Write(p []byte) (n int, err error) {
	l.Infof(string(p))
	return len(p), nil
}

// GetZapLogger returns the underlying zap.Logger for adapters that need it (e.g. Temporal).
func GetZapLogger() *zap.Logger {
	return zapLogger
}
