package hatchet

import (
	"encoding/json"
	"strings"

	"github.com/rs/zerolog"

	"getpaidhq/internal/core/port"
)

// newZerologToSlog builds a zerolog.Logger that forwards Hatchet SDK events
// to the app logger. The Hatchet client and worker only accept a
// *zerolog.Logger, so instead of letting them write their own JSON straight
// to stderr (`{"level":"debug","service":"client",...}` heartbeats every few
// seconds) we hand them a zerolog writing into a bridge that re-emits through
// slog. This unifies Hatchet's logs with the rest of the app's format.
//
// minLevel filters the SDK's chatter at the source, INDEPENDENTLY of the app
// log level (HATCHET_LOG_LEVEL) — so running the app at debug doesn't drown
// you in heartbeat noise, and you can crank the SDK to debug without touching
// the app level.
func newZerologToSlog(log port.Logger, minLevel string) zerolog.Logger {
	return zerolog.New(zerologSlogWriter{log: log}).Level(parseZerologLevel(minLevel))
}

// parseZerologLevel maps the HATCHET_LOG_LEVEL string onto zerolog levels.
// Empty or unknown values fall back to warn.
func parseZerologLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.WarnLevel
	}
}

// zerologSlogWriter is the io.Writer zerolog renders each event into. zerolog
// hands us a complete JSON object per Write; we decode it, lift level/message,
// and pass the remaining fields through as structured attributes.
type zerologSlogWriter struct {
	log port.Logger
}

func (w zerologSlogWriter) Write(p []byte) (int, error) {
	var fields map[string]any
	if err := json.Unmarshal(p, &fields); err != nil {
		// Not JSON (shouldn't happen with zerolog.New, but be safe) — emit raw.
		w.log.Info(strings.TrimSpace(string(p)))
		return len(p), nil
	}

	msg, _ := fields[zerolog.MessageFieldName].(string)
	level, _ := fields[zerolog.LevelFieldName].(string)
	delete(fields, zerolog.MessageFieldName)
	delete(fields, zerolog.LevelFieldName)
	delete(fields, zerolog.TimestampFieldName)

	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	switch level {
	case "trace", "debug":
		w.log.Debug(msg, attrs...)
	case "warn":
		w.log.Warn(msg, attrs...)
	case "error", "fatal", "panic":
		w.log.Error(msg, attrs...)
	default:
		w.log.Info(msg, attrs...)
	}

	return len(p), nil
}
