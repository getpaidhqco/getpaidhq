package hatchet

import (
	"encoding/json"
	"strings"

	"github.com/rs/zerolog"

	"getpaidhq/internal/core/port"
)

// newZerologToSlog builds a zerolog.Logger that forwards every event to the
// app logger. The Hatchet client and worker only accept a *zerolog.Logger, so
// instead of letting them write their own JSON straight to stderr
// (`{"level":"debug","service":"client",...}` heartbeats every few seconds) we
// hand them a zerolog writing into a bridge that re-emits through slog. This
// unifies Hatchet's logs with the rest of the app's format and level.
func newZerologToSlog(log port.Logger) zerolog.Logger {
	// DebugLevel here just means zerolog forwards everything to the bridge; the
	// app's slog handler applies the real level filter downstream.
	return zerolog.New(zerologSlogWriter{log: log}).Level(zerolog.DebugLevel)
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
