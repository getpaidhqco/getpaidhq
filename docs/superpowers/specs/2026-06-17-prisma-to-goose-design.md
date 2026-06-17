# Replace Prisma with Goose for schema management

**Date:** 2026-06-17
**Repo:** `gphq-server` (only — `gphq-web`/`gphq-checkout` do not use Prisma)
**Branch:** `worktree-prisma-to-goose` (off `main`)

## Goal

Retire Prisma as the schema/DDL tool in `gphq-server` and replace it with
[Goose](https://github.com/pressly/goose) SQL migrations. Generate a single
**baseline** migration per database that exactly reproduces today's
Prisma-managed schema, so future schema changes are hand-written, version-tracked
Goose migrations instead of `prisma db push`.

GORM remains the runtime query layer and is **not** touched. This is purely a
swap of the schema-management toolchain.

## Background / current state

- `gphq-server` is Go (1.26); GORM is the ORM for all queries. Prisma is used
  **only** to push DDL — it is never imported by Go.
- Three independent Postgres databases, each with its own Prisma schema and
  connection string:
  | DB | Schema file | URL env | Local DB name |
  | --- | --- | --- | --- |
  | operational ("app") | `schemas/app/schema.prisma` (1030 lines) | `DATABASE_URL` | `getpaidhq` |
  | reporting | `schemas/reporting/schema.prisma` (205 lines) | `REPORTING_DATABASE_URL` | `getpaidhq_reports` |
  | usage event store | `schemas/usage/schema.prisma` (43 lines) | `USAGE_DATABASE_URL` | `getpaidhq_usage` |
- Schema is pushed via `prisma db push` (no migration history). Makefile targets:
  `db-push`, `db-push-reporting`, `db-push-usage`, `db-push-all`, `db-format`.
- A SQL-migration convention already exists for ClickHouse:
  `internal/adapter/clickhouse/migrations/0001_meter_events.sql`.
- Integration tests (`internal/adapter/postgres/setup_test.go`, build tag
  `integration`) fake the schema with GORM `AutoMigrate(allModels()...)` because
  "Prisma can't be run from a Go test". This is a documented fidelity gap: no
  enums, FK constraints, defaults, or indexes.
- The platform currently runs local-only (pre-launch), so **fresh** databases are
  the dominant case.

## Decisions (locked)

1. **Execution model: CLI-only.** Goose runs as a pinned Go tool + Makefile
   targets. No `embed.FS`, no `migrate` subcommand in the server binary, no
   auto-migrate on startup.
2. **Integration tests run the real baseline.** Replace GORM `AutoMigrate` with
   applying the actual Goose baseline against the testcontainer, closing the
   fidelity gap.
3. **Full Prisma removal (clean break).** Delete `.prisma` files,
   `prisma.config.ts`, Prisma-coupled seed scripts, and Prisma deps. Goose SQL
   migrations become the only schema artifact.

## Design

### Migration layout

One Goose migrations directory per database (separate connection strings →
separate histories):

```
schemas/app/migrations/00001_baseline.sql
schemas/reporting/migrations/00001_baseline.sql
schemas/usage/migrations/00001_baseline.sql
```

Each baseline is a standard goose SQL file:

```sql
-- +goose Up
-- +goose StatementBegin
<full generated DDL>
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
<full reverse DDL — DROP everything the Up creates>
-- +goose StatementEnd
```

Sequential numbering (`00001_`), mirroring the existing ClickHouse convention.
Goose tracks applied versions in a `goose_db_version` table per database.

### Baseline generation (one-time, before Prisma is deleted)

For each schema, with Prisma still present, generate exact DDL:

```bash
# Up
prisma migrate diff --from-empty \
  --to-schema-datamodel schemas/<db>/schema.prisma --script
# Down
prisma migrate diff --to-empty \
  --from-schema-datamodel schemas/<db>/schema.prisma --script
```

Wrap each in goose annotations and write to the corresponding migrations dir.

**Verification gate (must pass before deleting Prisma):** for each DB, apply the
goose baseline to a throwaway Postgres and confirm zero drift versus a
`prisma db push`'d database. Method: spin two fresh databases, push Prisma to one
and `goose up` the other, then diff with `prisma migrate diff --from-url <goose-db>
--to-url <prisma-db> --script` — output must be empty. Only then proceed to
removal.

### Toolchain (CLI-only)

- Add goose as a pinned tool dependency: `go get -tool
  github.com/pressly/goose/v3/cmd/goose`. Invoked as `go tool goose` — no global
  install, version locked in `go.mod`.
- New Makefile targets (replacing all `db-push*` / `db-format`), per DB plus an
  `-all` aggregate. Each sets `GOOSE_DRIVER=postgres`,
  `GOOSE_DBSTRING=<the DB's URL>`, `GOOSE_MIGRATION_DIR=schemas/<db>/migrations`:
  - `db-migrate` / `db-migrate-reporting` / `db-migrate-usage` / `db-migrate-all` → `goose up`
  - `db-migrate-down` / `-reporting` / `-usage` → `goose down`
  - `db-migrate-status` / `-reporting` / `-usage` → `goose status`
  - `db-migrate-create name=<x>` → `goose -s create <x> sql` (scaffolds next file)
  - `db-seed` → `psql $DATABASE_URL -f schemas/app/seed.sql`

### Existing vs fresh databases

Fresh DBs (local, CI, testcontainers) run the baseline cleanly. For an existing
populated DB (the single pre-launch prod RDS) the baseline cannot `CREATE` over
live tables; document a **stamp runbook**: create `goose_db_version` and insert
`(version_id=1, is_applied=true)` so goose treats the baseline as already applied
and future migrations proceed. Manual, documented step — not automated (YAGNI).

### Integration tests

In `internal/adapter/postgres/setup_test.go`, replace
`db.AutoMigrate(allModels()...)` with applying the real baseline via the goose Go
library against the testcontainer:

```go
goose.SetDialect("postgres")
goose.Up(sqlDB, migrationsDir) // schemas/app/migrations, resolved from repo root
```

- `github.com/pressly/goose/v3` becomes a normal `require` (test imports it),
  coexisting with the `tool` directive.
- The migrations dir is resolved relative to the repo root (walk up from the test
  working dir to the module root, then `schemas/app/migrations`).
- Drop the now-inaccurate AutoMigrate comment block; the schema is now exact.

### Prisma removal (clean break)

Delete:
- `schemas/{app,reporting,usage}/schema.prisma`
- `schemas/{app,reporting,usage}/prisma.config.ts`
- `schemas/app/seed.js` and `schemas/reporting/seed_test.js` (Prisma-client
  coupled — confirmed `require('@prisma/client')` / `PrismaClient`)
- `prisma`, `@prisma/client`, `@prisma/adapter-pg` from `package.json`
  devDeps/deps; regenerate `pnpm-lock.yaml`; prune now-unused `node_modules`.

Keep:
- `schemas/app/seed.sql` (plain SQL) — wired to `make db-seed`.
- `package.json` itself — pruned to its non-Prisma scripts (tunnels, deploy,
  ngrok). If no JS deps remain after removal, drop the `dependencies`/
  `devDependencies` blocks accordingly.
- `docker/init/01-create-databases.sql` (creates the 3 local DBs).

### Docs & config updates

Update to describe the Goose flow instead of `prisma db push`:
- `Makefile` header comment + the "Database schema" section.
- `CLAUDE.md` (the "no migrations / clean db push only" guidance), `CONTEXT.md`,
  `README.md`, and `.env.example` schema-push comments.

## Out of scope

- `gphq-web` / `gphq-checkout` (no Prisma there).
- GORM query/model code.
- Embedding migrations in the binary / auto-migrate on startup (CLI-only chosen).
- Porting the faker-based seed scripts to SQL (they are deleted; re-authoring
  reporting/app dev fixtures in SQL is future work if needed).

## Verification (definition of done)

1. Each baseline applies cleanly to a fresh Postgres and shows **zero drift**
   versus `prisma db push` (the generation gate above).
2. `make db-migrate-all` succeeds against the local docker-compose stack.
3. `make test-integration` passes on the real-migration harness.
4. `go build ./...` and `go vet ./...` are clean.
5. No remaining references to `prisma` in `Makefile`, `package.json`, or Go code
   (`grep -ri prisma` shows only intentional historical mentions, if any).
```

