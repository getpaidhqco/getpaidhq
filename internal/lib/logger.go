package lib

import (
	"fmt"
	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"log/slog"
	"payloop/internal/application/lib/logger"
)

var (
	globalLogger logger.Logger
	zapLogger    *zap.Logger
)

type GinLogger struct {
	logger.Logger
}

type FxLogger struct {
	logger.Logger
}

// GetLogger get the logger
func GetLogger() logger.Logger {
	if globalLogger == nil {
		ll := newLogger(NewEnv(), zap.WithCaller(true))
		globalLogger = ll
	}
	return globalLogger
}

type MyLogger struct {
	logger *slog.Logger
}

// Implementing all methods of logger.Logger to MyLogger
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

func (l MyLogger) With(args ...interface{}) logger.Logger {
	return MyLogger{logger: l.logger.With(args...)}
}

func (l MyLogger) Sync() error {
	return nil
}

func (l MyLogger) Panic(args ...interface{}) {
	l.logger.Error("PANIC", args...)
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
	l.logger.Debug(fmt.Sprintf(template, args...))
}

// LogEvent log event for fx logger
func (l *FxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.Logger.Debug("OnStart hook executing: ",
			zap.String("callee", e.FunctionName),
			zap.String("caller", e.CallerName),
		)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.Logger.Debug("OnStart hook failed: ",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Error(e.Err),
			)
		} else {
			l.Logger.Debug("OnStart hook executed: ",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.String("runtime", e.Runtime.String()),
			)
		}
	case *fxevent.OnStopExecuting:
		l.Logger.Debug("OnStop hook executing: ",
			zap.String("callee", e.FunctionName),
			zap.String("caller", e.CallerName),
		)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.Logger.Debug("OnStop hook failed: ",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Error(e.Err),
			)
		} else {
			l.Logger.Debug("OnStop hook executed: ",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.String("runtime", e.Runtime.String()),
			)
		}
	case *fxevent.Supplied:
		l.Logger.Debug("supplied: ", slog.String("type", e.TypeName), slog.Any("err", e.Err))
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			l.Logger.Debug("provided: ", slog.String(e.ConstructorName, rtype))
		}
	case *fxevent.Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.Logger.Debug("decorated: ",
				zap.String("decorator", e.DecoratorName),
				zap.String("type", rtype),
			)
		}
	case *fxevent.Invoking:
		l.Logger.Debug("invoking: ", slog.String("func", e.FunctionName))
	case *fxevent.Started:
		if e.Err == nil {
			l.Logger.Debug("started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err == nil {
			l.Logger.Debug("initialized: custom fxevent.Logger -> ", slog.String("f", e.ConstructorName))
		}
	}
}

// newLogger sets up logger
func newLogger(env Env, opts ...zap.Option) logger.Logger {
	config := zap.NewDevelopmentConfig()
	logOutput := env.LogOutput

	if env.Env == "development" {
		fmt.Println("encode level")
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
	opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(4), zap.WithCaller(true), zap.AddStacktrace(zapcore.FatalLevel))
	config.Level.SetLevel(level)

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

// Write interface implementation for gin-framework
func (l GinLogger) Write(p []byte) (n int, err error) {
	l.Infof(string(p))
	return len(p), nil
}

// Printf prits go-fx logs
func (l FxLogger) Printf(str string, args ...interface{}) {
	if len(args) > 0 {
		l.Debugf(str, args)
	}
	l.Debug(str)
}

func GetZapLogger() *zap.Logger {
	return zapLogger
}

// GetFxLogger gets logger for go-fx
func GetFxLogger() fxevent.Logger {
	return &FxLogger{Logger: newLogger(NewEnv(), zap.WithCaller(false))}
}
