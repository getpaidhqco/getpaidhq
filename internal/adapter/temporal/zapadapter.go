package temporal

import (
	"fmt"
	"log/slog"

	temporallog "go.temporal.io/sdk/log"
)

// SlogAdapter adapts *slog.Logger to the Temporal SDK log.Logger interface.
type SlogAdapter struct {
	sl *slog.Logger
}

func NewSlogAdapter(sl *slog.Logger) *SlogAdapter {
	return &SlogAdapter{sl: sl}
}

func (a *SlogAdapter) attrs(keyvals []interface{}) []slog.Attr {
	if len(keyvals)%2 != 0 {
		return []slog.Attr{slog.String("error", fmt.Sprintf("odd number of keyvals: %v", keyvals))}
	}
	attrs := make([]slog.Attr, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", keyvals[i])
		}
		attrs = append(attrs, slog.Any(key, keyvals[i+1]))
	}
	return attrs
}

func (a *SlogAdapter) toArgs(keyvals []interface{}) []any {
	attrs := a.attrs(keyvals)
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	return args
}

func (a *SlogAdapter) Debug(msg string, keyvals ...interface{}) {
	a.sl.Debug(msg, a.toArgs(keyvals)...)
}

func (a *SlogAdapter) Info(msg string, keyvals ...interface{}) {
	a.sl.Info(msg, a.toArgs(keyvals)...)
}

func (a *SlogAdapter) Warn(msg string, keyvals ...interface{}) {
	a.sl.Warn(msg, a.toArgs(keyvals)...)
}

func (a *SlogAdapter) Error(msg string, keyvals ...interface{}) {
	a.sl.Error(msg, a.toArgs(keyvals)...)
}

func (a *SlogAdapter) With(keyvals ...interface{}) temporallog.Logger {
	return &SlogAdapter{sl: a.sl.With(a.toArgs(keyvals)...)}
}

func (a *SlogAdapter) WithCallerSkip(_ int) temporallog.Logger {
	// slog doesn't have caller skip; return as-is
	return a
}
