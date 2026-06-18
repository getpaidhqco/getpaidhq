# Logging

Everything logs through the injected `port.Logger`, backed by `log/slog`
(`internal/lib/logger.go`). Third-party output is bridged into the same
handler so one format and one set of level knobs covers the whole process:

- **GORM** → `internal/adapter/storage/postgresgorm/gorm_logger.go` (implements GORM's
  `logger.Interface` on top of `port.Logger`)
- **Hatchet SDK** (client + worker) → `internal/adapter/hatchet/zerolog_bridge.go`
  (a zerolog sink that re-emits through `port.Logger`)
- **Temporal SDK** → `temporallog.NewStructuredLogger(lib.GetSlogLogger())`
- **Fuego / anything using the default slog logger** → `slog.SetDefault` in
  `newLogger`

## Format

`LOG_FORMAT` selects the handler:

| Value    | Output                                                       |
| -------- | ------------------------------------------------------------ |
| `pretty` | colored single-line output via `lmittmann/tint`, trimmed source paths, `HH:MM:SS.mmm` timestamps — for humans |
| `text`   | logfmt (`slog.TextHandler`), no timestamps                    |
| `json`   | `slog.JSONHandler` — for log aggregation                      |

Unset defaults to `pretty` when `ENV` is `development` or `local`, `json`
otherwise.

## Levels — app vs infrastructure

The app level and the noisy infrastructure bridges are filtered
independently, so `GETPAIDHQ_LOG_LEVEL=debug` does NOT drag in SQL/heartbeat
spam, and vice versa:

| Var                   | Filters                          | Values                                  | Default |
| --------------------- | -------------------------------- | --------------------------------------- | ------- |
| `GETPAIDHQ_LOG_LEVEL` | the app (everything not below)   | `debug` `info` `warn` `error`           | `info`  |
| `GORM_LOG_LEVEL`      | SQL logs (filtered in the GORM bridge) | `silent` `error` `warn` `info`     | `warn` (slow queries + errors) |
| `HATCHET_LOG_LEVEL`   | Hatchet SDK chatter (filtered at the zerolog source) | `debug` `info` `warn` `error` | `warn` |

Caveat: the bridges *re-emit* through the app logger at the original
severity, so the app level still applies downstream. `GORM_LOG_LEVEL=info`
with `GETPAIDHQ_LOG_LEVEL=warn` shows nothing — open the app level first,
then the bridge level decides what's forwarded into it.

Typical local-dev setup (in `.env`):

```
ENV=local                  # → pretty format by default
GETPAIDHQ_LOG_LEVEL=debug  # your app, fully verbose
GORM_LOG_LEVEL=warn        # only slow queries + errors
HATCHET_LOG_LEVEL=warn     # no SDK heartbeats
```

## Conventions

- Inject `port.Logger`; never use `log`/`fmt` for runtime output.
- `MyLogger.emit` resolves the real caller for `source=` attribution (slog's
  own `AddSource` would otherwise always point at `lib/logger.go`).
- `LOG_OUTPUT` is only honored as a file path in production; everything else
  goes to stderr.
