# Why some DB logs look "unformatted"

## Symptom

Among the app's structured zap JSON logs, the occasional line looks completely different:

```
2026/06/03 18:28:40 /Users/.../internal/adapter/postgres/customer_repo.go:99
ERROR: ... (SQLSTATE 23503)
[3.236ms] [rows:0] INSERT INTO "customer_cohorts" (...) VALUES (...)
```

## Cause

That line comes from **GORM's own default logger**, not the app's `port.Logger`. In
`internal/adapter/postgres/database.go`:

```go
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info),   // ← GORM's built-in logger
})
```

- `logger.Default` writes with Go's stdlib `log` format (`2006/01/02 15:04:05 file:line …`), so it
  doesn't match the zap JSON the rest of the app emits.
- `LogMode(logger.Info)` logs **every** SQL statement (noisy), and renders errors in this stdlib
  shape.
- The rest of the app logs through `port.Logger` → zap (`internal/lib/logger.go`).

So GORM SQL/errors bypass zap entirely — that's why they look "unformatted."

## Fix (optional)

`NewDatabase(dsn string, log port.Logger)` already receives the app logger but doesn't use it. To
make DB logging consistent and quieter:

1. Implement a small adapter satisfying GORM's `logger.Interface` that delegates to `port.Logger`.
2. Pass it as `Logger:` in the `gorm.Config`.
3. Drop the level from `logger.Info` to `logger.Warn` (or `Error`) so normal SQL isn't logged,
   only slow queries / errors.

Not yet done — noted here so the stylistic inconsistency isn't mistaken for a bug.
