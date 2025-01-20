package lib

import (
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Logger structure
type Logger struct {
	*zap.SugaredLogger
}

type GinLogger struct {
	*Logger
}

type FxLogger struct {
	*Logger
}

var (
	globalLogger *Logger
	zapLogger    *zap.Logger
)

// GetLogger get the logger
func GetLogger() Logger {
	if globalLogger == nil {
		logger := newLogger()
		globalLogger = &logger
	}
	return *globalLogger
}

// GetGinLogger get the gin logger
func (l Logger) GetGinLogger() GinLogger {
	logger := zapLogger.WithOptions(
		zap.WithCaller(false),
	)
	return GinLogger{
		Logger: newSugaredLogger(logger),
	}
}

// GetFxLogger gets logger for go-fx
func (l *Logger) GetFxLogger() fxevent.Logger {
	logger := zapLogger.WithOptions(
		zap.WithCaller(false),
	)
	return &FxLogger{Logger: newSugaredLogger(logger)}
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
		l.Logger.Debug("supplied: ", zap.String("type", e.TypeName), zap.Error(e.Err))
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			l.Logger.Debug("provided: ", e.ConstructorName, " => ", rtype)
		}
	case *fxevent.Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.Logger.Debug("decorated: ",
				zap.String("decorator", e.DecoratorName),
				zap.String("type", rtype),
			)
		}
	case *fxevent.Invoking:
		l.Logger.Debug("invoking: ", e.FunctionName)
	case *fxevent.Started:
		if e.Err == nil {
			l.Logger.Debug("started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err == nil {
			l.Logger.Debug("initialized: custom fxevent.Logger -> ", e.ConstructorName)
		}
	}
}

func newSugaredLogger(logger *zap.Logger) *Logger {
	return &Logger{
		SugaredLogger: logger.Sugar(),
	}
}

// newLogger sets up logger
func newLogger() Logger {

	config := zap.NewDevelopmentConfig()

	level := zap.PanicLevel

	config.Level.SetLevel(level)

	zapLogger, _ = config.Build()
	logger := newSugaredLogger(zapLogger)

	return *logger
}

// Write interface implementation for gin-framework
func (l GinLogger) Write(p []byte) (n int, err error) {
	l.Info(string(p))
	return len(p), nil
}

// Printf prits go-fx logs
func (l FxLogger) Printf(str string, args ...interface{}) {
	if len(args) > 0 {
		l.Debugf(str, args)
	}
	l.Debug(str)
}
