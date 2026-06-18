# pgx Postgres adapter — design spec

**Status:** approved design, pre-implementation
**Date:** 2026-06-18
**Branch:** `worktree-pgx-adapter`

## Goal

Implement a complete, hand-written `jackc/pgx/v5` Postgres adapter alongside the
existing GORM adapter, selectable at runtime by an env var, with **100% behavioural
parity**. Default stays GORM, so the change is zero-risk to current behaviour until
`DB_DRIVER=pgx` is set.

"Parity" means: for every `port.*Repository` method (and `TxManager`, `EventStore`,
`PriorPaymentChecker`), both adapters produce the same observable result — same rows,
same domain values, same errors (`port.ErrNotFound`, unique/FK conflicts), same
transaction semantics — proven by running one shared integration suite against both.

## Non-goals

- No changes to `internal/core/{domain,port,service}` — the ports already abstract the
  data layer cleanly; this work must not touch them.
- No schema/migration work. Prisma→Goose is a separate, imminent PR; the pgx SQL simply
  targets the existing snake_case columns, which the Goose baseline reproduces exactly.
- No reporting revival. `report_repo.go` stays a stub in both adapters.
- No new business behaviour. This is a like-for-like port of the persistence layer.

## Background — what exists today

- **GORM adapter** at `internal/adapter/postgres/` (package `postgres`): 25 repo files
  (`<entity>_repo.go` + `<entity>_row.go`), plus `EventStore`, `PriorPaymentChecker`,
  `ReportRepo` (stub), `TxManager`, `NewDatabase`. 21 row types carry `serializer:json`
  columns, 3 use `shopspring/decimal`, 1 uses `SELECT … FOR UPDATE`
  (`subscription_repo.go`).
- Services depend only on `port.*Repository` interfaces; repos are constructed by hand in
  `internal/config/app.go` (`NewApp`) as `postgres.NewXxxRepo(db)`. This is the swap seam.
- **Transactions:** `port.TxManager.RunInTx(ctx, fn)` wraps `gorm.Transaction`, stashes the
  tx handle on `ctx` under a private `txKey{}`, and every repo reads it via
  `dbFromCtx(ctx, fallback)`. Post-commit side-effects (workflow starts, pubsub) run *after*
  `RunInTx` returns nil — never inside the closure. Row locking via `FindByIdForUpdate`.
- **Prior reference:** a fully-implemented pgx adapter lived at
  `internal/infrastructure/db/postgres/` before the hexagonal migration (commit
  `4ff4351^`): `pgxpool`, named-arg SQL (`@org_id`), a `PgDatabase` wrapping pool+tx,
  `getTransactionFromContext`, and a `mapError` translating pgconn error codes. It is a
  useful **reference** for the SQL/tx/error patterns but is not copy-pasteable (old
  `payloop` module path, uber/fx wiring, pre-hexagonal domain types).
- **Test isolation:** integration tests are `//go:build integration`, get a fresh
  `postgres:17-alpine` testcontainer via `testDB(t)` (`setup_test.go`), apply the schema
  baseline (`applyBaseline`), and scope rows with `uniqueOrg(t)` + `cleanupOrg(t, …)`. They
  currently live in `package postgres` and seed via **raw GORM** inserts.

## Transaction review (explicitly requested)

The current model is already adapter-agnostic and correct:

- `port.TxManager` is a one-method interface (`RunInTx`) that hides the engine.
- Tx handle travels on `ctx`; repos opt in via `dbFromCtx`. No per-request tx, no global
  state, no back-pointers.
- Post-commit side-effects are deliberately outside the tx closure so a rollback can't
  orphan a workflow start. First user: `OrderService.CompleteOrder`.
- `SELECT … FOR UPDATE` is available for state-transition flows.

**Verdict: optimal and unchanged.** pgx slots underneath the same port with no service,
port, or call-site changes. The pgx implementation mirrors GORM exactly:

- A package-private `querier` interface — `Exec`, `Query`, `QueryRow` — satisfied by both
  `*pgxpool.Pool` and `pgx.Tx`.
- `dbFromCtx(ctx, pool) querier` returns the ctx-stashed `pgx.Tx` if present, else the pool
  (mirrors GORM's `dbFromCtx`). Private `txKey{}` per package; only one adapter runs at a
  time, so there is no collision.
- `RunInTx`: if a tx is already on ctx → `tx.Begin(ctx)` (pgx creates a **savepoint**,
  matching GORM's nested-transaction-as-savepoint semantics); else `pool.Begin(ctx)`.
  `defer tx.Rollback(ctx)` (a post-commit rollback returns `ErrTxClosed`, which we ignore);
  commit on `fn` returning nil; on panic, rollback then re-panic (parity with
  `gorm.Transaction`).
- `FindByIdForUpdate` becomes a literal `… FOR UPDATE` SQL suffix — simpler than GORM's
  clause builder.

## Architecture

### New package

`internal/adapter/postgrespgx/` (package `postgrespgx`), mirroring `postgres/`
file-for-file. Each repo exposes the **same constructor signature and return type** as the
GORM one, e.g. `func NewCustomerRepo(pool *pgxpool.Pool) port.CustomerRepository`. The GORM
package is left untouched.

Files:

- `database.go` — `NewDatabase(dsn, log, logLevel) (*pgxpool.Pool, error)`; pool tuning
  mirrors the four GORM knobs (`MaxConns`/`MinConns`/`MaxConnLifetime`/`MaxConnIdleTime`)
  via `pgxpool.Config`; optional query tracer wired to `port.Logger` gated by log level.
- `tx.go` — `querier`, `txKey`, `WithTx`, `dbFromCtx`, `TxManager` (+ `var _ port.TxManager`).
- `errs.go` — `translateErr` (→ `port.ErrNotFound`) and the pgconn-code mapping
  (`23505`/`23503`/`23502`/`42P01`) plus the `asConflictOnUnique` helper used on writes.
- `scopes.go` — pagination clause builder + identifier allowlist (`^[a-z_][a-z0-9_]*$`),
  `[1,200]` limit clamp, default `created_at DESC` — same rules as the GORM `Paginate`.
- `row_helpers.go` — `strOrEmpty`/`nilIfEmpty`/`emptyIfNil`, `nullTime`, and a generic
  `jsonCol[T]` implementing `Scan`/`Value` for the 21 JSON columns; `decimal.Decimal` and
  `*string` cover the rest.
- `<entity>_row.go` — row structs tagged `db:"col"`; the existing `toDomain`/`fromDomain`
  mappers carried over (domain types unchanged).
- `<entity>_repo.go` — hand-written parameterized SQL (`$1`…), explicit `WHERE org_id = $1`
  scoping; reads via `pgx.CollectRows` + `pgx.RowToAddrOfStructByName`; list+count as two
  statements.
- `event_store.go`, `prior_payment_checker.go`, `report_repo.go` (stub).

### Selection & wiring

- New env `DB_DRIVER` (`gorm` default | `pgx`) added to `lib.Env`, `viper.BindEnv` in
  `NewEnv()`, and `.env.example`.
- In `app.go`, introduce a `repoSet` struct holding every `port.*Repository` + `TxManager`
  + `EventStore` + `PriorPaymentChecker`, with two builders `newGormRepoSet(...)` /
  `newPgxRepoSet(...)` selected by `env.DBDriver`. The rest of `NewApp` consumes `repoSet`
  (all port types) unchanged.
- `App.DB *gorm.DB` is generalised to a small driver-agnostic seam (health-ping + close)
  so the field no longer hard-codes GORM. (Audit `App.DB` consumers during implementation;
  expected to be shutdown/health only.)
- `usageDB`/`buildEventStore` are parameterised by driver the same way (the usage store is
  in scope — full surface).

## Row mapping & SQL conventions

- Reads scan into row structs via `pgx.RowToAddrOfStructByName` (`db` tags) and
  `pgx.CollectRows`; single-row lookups use `QueryRow` + `pgx.RowToStructByName`, mapping
  `pgx.ErrNoRows` → `port.ErrNotFound`.
- JSON columns go through `jsonCol[T]` (marshal on `Value`, unmarshal on `Scan`); NOT NULL
  JSON columns receive `{}` via `emptyIfNil`, matching GORM.
- Nullable FK/unique ids written as SQL `NULL` (never `""`) — same rule as the recent GORM
  fix (e.g. `customers.default_payment_method_id`, `external_id`): `nilIfEmpty` on write.
- Pagination/sorting reuse the allowlist + clamp; org scoping is always an explicit
  predicate.
- Error semantics: writes that hit a unique index return the same conflict error the GORM
  repo returns (`asConflictOnUnique` with the same message), so handler/service behaviour is
  identical.

## Parity test harness (the load-bearing piece)

The existing integration/e2e tests live in `package postgres` and seed via raw GORM, so
they cannot run unmodified against pgx. Plan:

1. Extract the integration + e2e billing tests into a **driver-agnostic conformance
   suite** under `internal/repotest/` — a top-level package (NOT an adapter, NOT core)
   that verifies any implementation of the repository ports behaves identically. It
   receives (a) a `RepoSet` factory and (b) a pool/handle, and contains all assertions.
   It depends only on `domain` + `port` + the testcontainer setup; adapters import it,
   never the reverse (the Go "conformance suite beside the contract" pattern, cf.
   `fstest`/`iotest`). It is kept out of `internal/core/port/` so the testcontainer
   dependency never lands under the deliberately-pure core.
2. **Rewrite seed helpers to go through repo `Create` methods** (and `TxManager`) instead
   of raw row inserts — they already exercise the domain mappers, so this removes the
   GORM coupling without weakening coverage.
3. `postgres` and `postgrespgx` each get a thin `*_integration_test.go` that invokes the
   shared suite with their own factory. Same testcontainer + same schema baseline.
4. CI grows a driver dimension so `make test-integration` (and the race job) runs the suite
   under both `gorm` and `pgx`. A `TEST_DB_DRIVER` env (or build matrix) selects which.

This shared suite is what actually proves "100% parity" — the same assertions, both
adapters. It is the largest single chunk of work and is sequenced before the bulk of the
repo implementations so each repo cluster is verified against it as it lands.

## Scope & phasing

Full surface. Implemented and verified in clusters:

1. **Foundation** — `database.go` (pool), `tx.go`, `errs.go`, `scopes.go`, scan helpers;
   `DB_DRIVER` env + `repoSet` refactor in `app.go` with **GORM still default**. App builds
   and runs exactly as before.
2. **Harness** — extract the driver-agnostic conformance suite (`internal/repotest/`),
   reseed via ports, get the GORM suite green through it (proves the harness is faithful
   before pgx exists).
3. **Repos, FK-dependency order** — org → customer (+cohort) → product → variant → price →
   order (+items) → subscription (incl. `FindByIdForUpdate`, `FindDueForBilling`,
   `FindUpcomingRenewals`, `FindActiveMeteredForMeter`) → payment (+refund) → payment_method
   → invoice (+line items, atomic create) → dunning (campaign/attempt/config/communication/
   token/history) → coupon → coupon_code → discount → setting → psp → api_key → idempotency
   (atomic `Claim`/`Release`) → metadata → session → cart → webhook_subscription → meter.
   Run the shared suite against pgx after each cluster.
4. **Usage & misc** — `EventStore` (incl. the filter-group query builder — the most complex
   hand-written SQL) + `PriorPaymentChecker` + `report_repo.go` stub.
5. **CI** — flip the integration job to the driver matrix; document `DB_DRIVER` in
   `.env.example` and the relevant CLAUDE.md/docs.

## Risks & mitigations

- **Test-harness refactor churn** (largest piece): mitigated by reseeding through ports
  (mappers already covered) and landing it in phase 2, before pgx repos, with the GORM
  suite as the fidelity check.
- **JSON/decimal/enum scanning subtleties**: centralised in `row_helpers.go` + `jsonCol[T]`
  so each repo doesn't re-solve them; verified by the shared suite's round-trip assertions.
- **Nested `RunInTx`**: handled by ctx-tx detection + pgx savepoints to match GORM; covered
  by a harness test that nests transactions and asserts savepoint rollback behaviour.
- **`EventStore` filter-group SQL**: the highest-complexity translation; its existing
  integration tests (`event_store_filter_group_integration_test.go`) move into the shared
  suite and gate the pgx implementation.
- **`App.DB` coupling**: audited and replaced with a driver-agnostic seam in phase 1.

## Acceptance criteria

- `DB_DRIVER=pgx` boots the full app (operational + usage paths) with no GORM dependency on
  the request path.
- The shared integration + e2e suite passes identically under both `gorm` and `pgx` in CI.
- `DB_DRIVER` unset/`gorm` is byte-for-byte the current behaviour.
- No changes to `core/{domain,port,service}`.
