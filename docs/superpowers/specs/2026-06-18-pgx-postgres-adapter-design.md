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
transaction semantics — proven by running **one shared conformance suite against both**.

## Non-goals

- No changes to `internal/core/{domain,port,service}` — the ports already abstract the
  data layer cleanly; this work must not touch them. Ports stay in `internal/core/port`;
  adapters import ports, ports never import adapters (the storage restructure below does
  not change this dependency direction).
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
  currently live in `package postgres` and seed via **raw GORM** inserts — i.e. they are
  welded to the GORM adapter and cannot run against another implementation as-is.

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

### Storage restructure (category-nested layout)

The storage adapters move under a category package, with a shared test-helper package
beside them:

```
internal/adapter/storage/
  postgresgorm/   ← today's internal/adapter/postgres, renamed (package postgres → postgresgorm)
  postgrespgx/    ← new pgx adapter (package postgrespgx — not "pgx", which collides with the jackc import)
  storagetest/    ← test-helper package: exports the conformance suite the adapters call
```

This is idiomatic Go (`internal/adapter/<category>/<impl>`). The category package gives the
shared conformance suite a natural home. Nesting does not change dependency direction:
both adapters import `internal/core/port`; nothing imports the adapters except `app.go` and
each adapter's own `_test.go`.

Each adapter exposes the **same constructor signatures and return types** as the GORM one
does today, e.g. `func NewCustomerRepo(pool *pgxpool.Pool) port.CustomerRepository`.

pgx package files (mirroring `postgresgorm/` file-for-file):

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

## Parity proof — the conformance suite

The existing integration/e2e tests live in `package postgres` and seed via raw GORM, so
they cannot run unmodified against pgx. They become a **driver-agnostic conformance suite**
in `internal/adapter/storage/storagetest`, which each adapter's own `_test.go` calls with
its own instances. This is the test-helper-package pattern (not a parent package reaching
down into its children — that would invert the import direction).

`storagetest` owns:

1. **Infra** — the `postgres:17-alpine` testcontainer + Goose baseline + per-test
   `uniqueOrg`/`cleanupOrg`, all driver-agnostic (they only need a DSN).
2. **A `RepoSet`** — a struct of every `port.*Repository` + `TxManager` + `EventStore` +
   `PriorPaymentChecker`, plus a factory type `func(t *testing.T, dsn string) RepoSet`.
3. **Seed helpers rewritten to go through `RepoSet` `Create` methods** (and `TxManager`)
   instead of raw row inserts — they already exercise the domain mappers, so this removes
   the GORM coupling without weakening coverage.
4. **The assertions** — exported as `RunConformance(t, factory)` (sub-tests cover every
   repo method + the e2e billing/usage flows).

Each adapter then has a one-line entrypoint, e.g. in `postgresgorm`:

```go
//go:build integration
func TestConformance(t *testing.T) { storagetest.RunConformance(t, postgresgorm.NewRepoSet) }
```

and the identical thing in `postgrespgx`. No import cycle: `storagetest` imports only
`domain` + `port`; the adapters import `storagetest` from their test files. CI runs both,
so the same assertions gate both adapters — this is what actually proves "100% parity".

## Scope & phasing

Full surface. Sequenced so the suite exists and is trusted before pgx is written, and each
pgx cluster is verified against it as it lands.

0. **Restructure** — move `internal/adapter/postgres` → `internal/adapter/storage/postgresgorm`
   (rename package `postgres` → `postgresgorm`), update `app.go` + imports + the living
   docs that name the path (atomic with the move; see Documentation updates). GORM remains
   the only driver and the default. Build + full test suite green. Standalone commit so the
   diff is obviously a pure relocation.
1. **Conformance suite** — extract `internal/adapter/storage/storagetest` (infra + `RepoSet`
   + ports-based seeds + `RunConformance`); repoint `postgresgorm`'s integration tests at
   it. Get the GORM suite green through it (proves the suite is faithful before pgx exists).
2. **pgx foundation** — `database.go` (pool), `tx.go`, `errs.go`, `scopes.go`, scan helpers;
   `DB_DRIVER` env + `repoSet` selection in `app.go` with **GORM still default**. App builds
   and runs exactly as before.
3. **pgx repos, FK-dependency order** — org → customer (+cohort) → product → variant →
   price → order (+items) → subscription (incl. `FindByIdForUpdate`, `FindDueForBilling`,
   `FindUpcomingRenewals`, `FindActiveMeteredForMeter`) → payment (+refund) → payment_method
   → invoice (+line items, atomic create) → dunning (campaign/attempt/config/communication/
   token/history) → coupon → coupon_code → discount → setting → psp → api_key → idempotency
   (atomic `Claim`/`Release`) → metadata → session → cart → webhook_subscription → meter.
   Run `RunConformance` against pgx after each cluster.
4. **pgx usage & misc** — `EventStore` (incl. the filter-group query builder — the most
   complex hand-written SQL) + `PriorPaymentChecker` + `report_repo.go` stub.
5. **CI & docs** — add the driver dimension so `make test-integration` (and the race job)
   runs `RunConformance` under both `gorm` and `pgx`; finalize docs.

## Documentation updates

The restructure changes a live path (`internal/adapter/postgres/…`), so the **living** docs
that reference it are updated atomically with the phase-0 move:

- `gphq-server/CLAUDE.md` — Architecture section (state the `internal/adapter/<category>/<impl>`
  layout + the `storagetest` conformance convention + `DB_DRIVER`) and the Test-database
  isolation section (suite now lives in `storagetest`, runs against both adapters).
- `docs/internal/hexagonal-mapping-pattern.md` — the layer diagram, the "Postgres row" table
  row, and the "where to add a new row/repo" steps.
- `docs/architecture/system-hexagonal.md` — path references.
- `docs/internal/logging.md` — the `gorm_logger.go` path.
- `docs/workflows/*` — inline `internal/adapter/postgres/<file>` references.

**Explicitly NOT updated:** `docs/superpowers/plans/*` and `docs/superpowers/specs/*` from
prior dated work. They are point-in-time records of shipped changes; rewriting their paths
would falsify history. They keep referencing `internal/adapter/postgres/` as it was when
written.

## Risks & mitigations

- **Phase-0 relocation churn**: large mechanical move. Mitigated by doing it as a pure
  rename in its own commit, GORM-only/default throughout, with the full existing suite green
  before and after, and living-doc path updates in the same commit.
- **Conformance-suite extraction**: the second-largest piece. Mitigated by reseeding through
  ports (mappers already covered) and landing it in phase 1, before pgx repos, with the GORM
  suite as the fidelity check.
- **JSON/decimal/enum scanning subtleties**: centralised in `row_helpers.go` + `jsonCol[T]`
  so each repo doesn't re-solve them; verified by the suite's round-trip assertions.
- **Nested `RunInTx`**: handled by ctx-tx detection + pgx savepoints to match GORM; covered
  by a suite test that nests transactions and asserts savepoint rollback behaviour.
- **`EventStore` filter-group SQL**: the highest-complexity translation; its existing
  integration tests (`event_store_filter_group_integration_test.go`) move into the suite and
  gate the pgx implementation.
- **`App.DB` coupling**: audited and replaced with a driver-agnostic seam in phase 2.

## Acceptance criteria

- `DB_DRIVER=pgx` boots the full app (operational + usage paths) with no GORM dependency on
  the request path.
- `storagetest.RunConformance` passes identically under both `gorm` and `pgx` in CI.
- `DB_DRIVER` unset/`gorm` is byte-for-byte the current behaviour.
- No changes to `core/{domain,port,service}`.
