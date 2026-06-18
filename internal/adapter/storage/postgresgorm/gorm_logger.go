package postgresgorm

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	gormutils "gorm.io/gorm/utils"

	"getpaidhq/internal/core/port"
)

// gormSlogLogger adapts GORM's logger.Interface onto our port.Logger so that
// SQL logs share the application's slog format and level. GORM's stock logger
// writes its own stdlib-style lines straight to stdout
// (`2026/... event_store.go:35 [1.090ms] CREATE INDEX ...`), which is why DB
// logs looked nothing like the rest of the app.
type gormSlogLogger struct {
	log           port.Logger
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

// newGormLogger builds the adapter. slowThreshold mirrors GORM's default
// (200ms) — queries slower than this log at Warn even in otherwise-quiet modes.
func newGormLogger(log port.Logger, level gormlogger.LogLevel) gormSlogLogger {
	return gormSlogLogger{log: log, level: level, slowThreshold: 200 * time.Millisecond}
}

func (l gormSlogLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	l.level = level
	return l
}

func (l gormSlogLogger) Info(_ context.Context, msg string, data ...any) {
	if l.level >= gormlogger.Info {
		l.log.Infof(msg, data...)
	}
}

func (l gormSlogLogger) Warn(_ context.Context, msg string, data ...any) {
	if l.level >= gormlogger.Warn {
		l.log.Warnf(msg, data...)
	}
}

func (l gormSlogLogger) Error(_ context.Context, msg string, data ...any) {
	if l.level >= gormlogger.Error {
		l.log.Errorf(msg, data...)
	}
}

func (l gormSlogLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.level >= gormlogger.Error && !errors.Is(err, gorm.ErrRecordNotFound):
		l.log.Error("gorm query failed",
			"err", err.Error(), "caller", gormutils.FileWithLineNum(),
			"elapsed_ms", elapsed.Milliseconds(), "rows", rows, "sql", sql)
	case elapsed > l.slowThreshold && l.level >= gormlogger.Warn:
		l.log.Warn("gorm slow query",
			"caller", gormutils.FileWithLineNum(),
			"elapsed_ms", elapsed.Milliseconds(), "rows", rows, "sql", sql)
	case l.level >= gormlogger.Info:
		l.log.Info("gorm query",
			"caller", gormutils.FileWithLineNum(),
			"elapsed_ms", elapsed.Milliseconds(), "rows", rows, "sql", sql)
	}
}
