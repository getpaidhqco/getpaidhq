# Prisma → Goose Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Prisma `db push` schema management in `gphq-server` with Goose SQL migrations, seeded by an exact baseline migration per database.

**Architecture:** Generate a baseline `.sql` migration per DB from the current Prisma schemas (verified zero-drift), wire Goose as a pinned Go tool + Makefile targets, switch integration tests to apply the real baseline, then delete all Prisma artifacts.

**Tech Stack:** Go 1.26, Goose v3 (`github.com/pressly/goose/v3`), Postgres 17, GORM (query layer, untouched), Testcontainers, Prisma 7.8 (used once to generate baselines, then removed).

**Working dir:** `gphq-server` worktree on branch `worktree-prisma-to-goose`. All paths below are relative to the `gphq-server` repo root.

**Reference spec:** `docs/superpowers/specs/2026-06-17-prisma-to-goose-design.md`

---

## File Structure

**Create:**
- `schemas/app/migrations/00001_baseline.sql` — operational DB baseline
- `schemas/reporting/migrations/00001_baseline.sql` — reporting DB baseline
- `schemas/usage/migrations/00001_baseline.sql` — usage DB baseline
- `internal/adapter/postgres/migrate_test.go` — repo-root + goose helper for tests (test-only, `integration` tag)

**Modify:**
- `go.mod` / `go.sum` — add goose tool directive + library require
- `Makefile` — replace `db-push*`/`db-format` with `db-migrate*`/`db-seed`
- `internal/adapter/postgres/setup_test.go` — apply real baseline instead of AutoMigrate
- `package.json` — drop Prisma scripts + deps
- `CLAUDE.md`, `CONTEXT.md`, `README.md`, `.env.example` — describe Goose flow

**Delete:**
- `schemas/{app,reporting,usage}/schema.prisma`
- `schemas/{app,reporting,usage}/prisma.config.ts`
- `schemas/app/seed.js`, `schemas/reporting/seed_test.js`
- `pnpm-lock.yaml` (regenerated), Prisma entries in `node_modules`

---

## Task 1: Add Goose as a pinned tool + library dependency

**Files:** Modify `go.mod`, `go.sum`

- [ ] **Step 1: Add the goose CLI as a Go tool dependency**

Run:
```bash
go get -tool github.com/pressly/goose/v3/cmd/goose@latest
```
Expected: `go.mod` gains a `tool github.com/pressly/goose/v3/cmd/goose` line and a matching `require`.

- [ ] **Step 2: Add the goose library require (used by integration tests)**

Run:
```bash
go get github.com/pressly/goose/v3@latest
```
Expected: `github.com/pressly/goose/v3` present in `go.mod` `require` (not `// indirect`).

- [ ] **Step 3: Verify the CLI is invokable via the tool directive**

Run:
```bash
go tool goose --version
```
Expected: prints `goose version: v3.x.x` (no "unknown tool" error).

- [ ] **Step 4: Tidy**

Run:
```bash
go mod tidy
```
Expected: clean exit, no errors.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit -m "build: add goose v3 as pinned tool + library dependency"
```

---

## Task 2: Generate the three baseline migrations from Prisma

Prisma 7.8 is still present here, so `prisma migrate diff` can emit exact DDL. The `--from-empty`/`--to-schema-datamodel` form computes purely from the schema file and does **not** require a live database connection.

**Files:** Create `schemas/{app,reporting,usage}/migrations/00001_baseline.sql`

- [ ] **Step 1: Create the migrations directories**

Run:
```bash
mkdir -p schemas/app/migrations schemas/reporting/migrations schemas/usage/migrations
```

- [ ] **Step 2: Generate the app baseline Up + Down DDL**

Run:
```bash
pnpm exec prisma migrate diff --from-empty \
  --to-schema-datamodel schemas/app/schema.prisma --script > /tmp/app_up.sql
pnpm exec prisma migrate diff --to-empty \
  --from-schema-datamodel schemas/app/schema.prisma --script > /tmp/app_down.sql
```
Expected: `/tmp/app_up.sql` contains many `CREATE TABLE`/`CREATE TYPE`/`CREATE INDEX`/`ALTER TABLE ... ADD CONSTRAINT` statements; `/tmp/app_down.sql` contains the corresponding `DROP` statements.
If `migrate diff` errors asking for a config, retry adding `--config schemas/app/prisma.config.ts`.

- [ ] **Step 3: Assemble `schemas/app/migrations/00001_baseline.sql`**

Write the file with this exact structure (do NOT wrap the bulk DDL in `StatementBegin/End` — goose splits on `;`, which is correct for these plain table/enum/index/FK statements):

```sql
-- +goose Up
-- Baseline generated from schemas/app/schema.prisma via `prisma migrate diff`.
-- Reproduces the operational schema exactly as of the Prisma→Goose cutover.

<contents of /tmp/app_up.sql>

-- +goose Down

<contents of /tmp/app_down.sql>
```

- [ ] **Step 4: Generate + assemble the reporting baseline**

Run:
```bash
pnpm exec prisma migrate diff --from-empty \
  --to-schema-datamodel schemas/reporting/schema.prisma --script > /tmp/rep_up.sql
pnpm exec prisma migrate diff --to-empty \
  --from-schema-datamodel schemas/reporting/schema.prisma --script > /tmp/rep_down.sql
```
Then write `schemas/reporting/migrations/00001_baseline.sql` using the same `-- +goose Up` / `-- +goose Down` structure as Step 3, with the reporting source comment.

- [ ] **Step 5: Generate + assemble the usage baseline**

Run:
```bash
pnpm exec prisma migrate diff --from-empty \
  --to-schema-datamodel schemas/usage/schema.prisma --script > /tmp/use_up.sql
pnpm exec prisma migrate diff --to-empty \
  --from-schema-datamodel schemas/usage/schema.prisma --script > /tmp/use_down.sql
```
Then write `schemas/usage/migrations/00001_baseline.sql` with the same structure and the usage source comment.

- [ ] **Step 6: Sanity-check the goose annotations are present in all three**

Run:
```bash
grep -L "+goose Up" schemas/*/migrations/00001_baseline.sql; \
grep -L "+goose Down" schemas/*/migrations/00001_baseline.sql
```
Expected: no output (every file contains both markers).

- [ ] **Step 7: Commit**

```bash
git add schemas/app/migrations schemas/reporting/migrations schemas/usage/migrations
git commit -m "feat(db): add Goose baseline migrations generated from Prisma schemas"
```

---

## Task 3: Verify baselines are zero-drift vs Prisma (GATE — must pass before Prisma removal)

This proves the goose baseline produces the identical schema Prisma would. Runs against the local docker-compose Postgres.

**Files:** none (verification only)

- [ ] **Step 1: Start local Postgres**

Run:
```bash
make up
```
Expected: docker-compose stack up; Postgres healthy. Note the operational `DATABASE_URL` from `.env` (e.g. `postgres://getpaidhq:getpaidhq@localhost:10432/getpaidhq?sslmode=disable`).

- [ ] **Step 2: Create two scratch databases on the same instance**

Using the same host/port/creds as `DATABASE_URL`:
```bash
PGBASE="postgres://getpaidhq:getpaidhq@localhost:10432"
psql "$PGBASE/getpaidhq?sslmode=disable" -c "DROP DATABASE IF EXISTS drift_goose; CREATE DATABASE drift_goose;"
```
Expected: `CREATE DATABASE`.

- [ ] **Step 3: Apply the app baseline to `drift_goose` with goose**

Run:
```bash
GOOSE_DRIVER=postgres \
GOOSE_DBSTRING="$PGBASE/drift_goose?sslmode=disable" \
GOOSE_MIGRATION_DIR=schemas/app/migrations \
go tool goose up
```
Expected: `OK   00001_baseline.sql` and `goose: successfully migrated database`.

- [ ] **Step 4: Drop goose's bookkeeping table so it doesn't show as drift**

Run:
```bash
psql "$PGBASE/drift_goose?sslmode=disable" -c "DROP TABLE goose_db_version;"
```
Expected: `DROP TABLE`.

- [ ] **Step 5: Diff the goose-applied DB against the Prisma schema datamodel**

Run:
```bash
pnpm exec prisma migrate diff \
  --from-url "$PGBASE/drift_goose?sslmode=disable" \
  --to-schema-datamodel schemas/app/schema.prisma --script
```
Expected: **empty output** (no SQL) = the goose baseline reproduces the Prisma schema exactly. If non-empty, the printed statements are the drift — fix `schemas/app/migrations/00001_baseline.sql` and repeat from Step 2.

- [ ] **Step 6: Repeat the gate for reporting and usage**

Repeat Steps 2–5 with `drift_goose` recreated each time, `GOOSE_MIGRATION_DIR=schemas/reporting/migrations` then `schemas/usage/migrations`, diffing against `schemas/reporting/schema.prisma` then `schemas/usage/schema.prisma`. Each must produce empty diff output.

- [ ] **Step 7: Clean up scratch DB**

Run:
```bash
psql "$PGBASE/getpaidhq?sslmode=disable" -c "DROP DATABASE IF EXISTS drift_goose;"
```

No commit (verification only). Do not proceed to Task 6 until all three diffs are empty.

---

## Task 4: Wire Makefile targets (Goose in, Prisma push out)

**Files:** Modify `Makefile`

- [ ] **Step 1: Replace the "Database schema" section**

In `Makefile`, delete the `db-push`, `db-push-reporting`, `db-push-usage`, `db-push-all`, and `db-format` targets and the `## ---- Database schema (Prisma db push — no migrations) ----` header. Insert this section in their place:

```makefile
## ---- Database schema (Goose migrations) ------------------------------------

GOOSE        := go tool goose
GOOSE_DRIVER := postgres

# $(call goose,<migration-dir>,<dbstring>,<args...>)
define goose
	GOOSE_DRIVER=$(GOOSE_DRIVER) GOOSE_DBSTRING="$(2)" GOOSE_MIGRATION_DIR=$(1) $(GOOSE) $(3)
endef

.PHONY: db-migrate
db-migrate: ## Apply operational migrations -> DATABASE_URL
	$(call goose,schemas/app/migrations,$(DATABASE_URL),up)

.PHONY: db-migrate-reporting
db-migrate-reporting: ## Apply reporting migrations -> REPORTING_DATABASE_URL
	$(call goose,schemas/reporting/migrations,$(REPORTING_DATABASE_URL),up)

.PHONY: db-migrate-usage
db-migrate-usage: ## Apply usage-store migrations -> USAGE_DATABASE_URL
	$(call goose,schemas/usage/migrations,$(USAGE_DATABASE_URL),up)

.PHONY: db-migrate-all
db-migrate-all: db-migrate db-migrate-reporting db-migrate-usage ## Apply all three schemas

.PHONY: db-migrate-status
db-migrate-status: ## Show operational migration status
	$(call goose,schemas/app/migrations,$(DATABASE_URL),status)

.PHONY: db-migrate-down
db-migrate-down: ## Roll back the last operational migration
	$(call goose,schemas/app/migrations,$(DATABASE_URL),down)

.PHONY: db-migrate-create
db-migrate-create: ## Scaffold a new operational migration: make db-migrate-create name=add_foo
	$(call goose,schemas/app/migrations,$(DATABASE_URL),create $(name) sql)

.PHONY: db-seed
db-seed: ## Seed the operational DB from schemas/app/seed.sql
	psql "$(DATABASE_URL)" -f schemas/app/seed.sql
```

Note: `DATABASE_URL` etc. come from the environment / `.env`. If the Makefile does not already load `.env`, prefix invocations with `set -a; . ./.env; set +a;` or run targets with the vars exported (document in README, Task 7).

- [ ] **Step 2: Update the Makefile header comment**

In the top comment block, change the `pnpm scripts (Prisma schema push, ...)` line to read `pnpm scripts (tunnels, deploy — see package.json)` and remove the Prisma mention.

- [ ] **Step 3: Verify targets parse and run**

Run (with env loaded):
```bash
make db-migrate-status
```
Expected: goose prints the status table for `schemas/app/migrations` against `DATABASE_URL` (lists `00001_baseline.sql`).

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "build(make): replace prisma db-push targets with goose db-migrate targets"
```

---

## Task 5: Switch integration tests to apply the real baseline

Replace GORM `AutoMigrate` with the goose baseline so tests run against the exact production schema (enums, FKs, indexes, defaults).

**Files:** Create `internal/adapter/postgres/migrate_test.go`; Modify `internal/adapter/postgres/setup_test.go`

- [ ] **Step 1: Write the migrations helper (test-only)**

Create `internal/adapter/postgres/migrate_test.go`:

```go
//go:build integration

package postgres

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// repoRoot walks up from the current working directory until it finds go.mod,
// returning the module root. Integration tests run from the package dir, so the
// migrations live at <root>/schemas/app/migrations.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "reached filesystem root without finding go.mod")
		dir = parent
	}
}

// applyBaseline runs the operational Goose migrations against db.
func applyBaseline(t *testing.T, db *sql.DB) {
	t.Helper()
	require.NoError(t, goose.SetDialect("postgres"))
	dir := filepath.Join(repoRoot(t), "schemas", "app", "migrations")
	require.NoError(t, goose.Up(db, dir))
}
```

- [ ] **Step 2: Replace AutoMigrate in `setup_test.go`**

In `internal/adapter/postgres/setup_test.go`, find the `sharedOnce.Do` body where it opens the GORM connection and calls `db.AutoMigrate(allModels()...)`. Replace the AutoMigrate call with:

```go
	sqlDB, err := db.DB()
	if err != nil {
		sharedErr = fmt.Errorf("failed to get *sql.DB: %w", err)
		return
	}
	applyBaseline(t, sqlDB) // applies schemas/app/migrations baseline
```

The AutoMigrate call sits inside `sharedOnce.Do(func() {...})`, which has no `*testing.T` parameter. Capture the `t` from `testDB(t)` before the closure so it is in scope: add `tCaptured := t` immediately before `sharedOnce.Do(func() {`, and inside the closure call `applyBaseline(tCaptured, sqlDB)`. (The closure already closes over outer variables like `sharedErr`, so this is the same pattern.) Then remove the now-unused `allModels()` function and any imports it solely required (check with `grep -rn allModels internal/adapter/postgres`).

- [ ] **Step 3: Update the stale schema comment**

Replace the `Schema:` paragraph in the `setup_test.go` header comment (the block explaining the AutoMigrate caveat) with:

```go
// Schema: each container has the real operational baseline applied via Goose
// (schemas/app/migrations), so enums, FK constraints, defaults, and indexes
// match production exactly.
```

- [ ] **Step 4: Build the integration test binary (compile check)**

Run:
```bash
go test -tags=integration -run xxxNoSuchTest ./internal/adapter/postgres/...
```
Expected: compiles and runs 0 tests (`ok` / `no tests to run`), proving the goose wiring builds.

- [ ] **Step 5: Run the postgres integration tests**

Run (Docker must be available for Testcontainers):
```bash
go test -tags=integration ./internal/adapter/postgres/...
```
Expected: PASS. If a repo test fails because a column/constraint now differs from what AutoMigrate previously faked, that is a real schema-fidelity finding — fix the test's expectations, not the baseline.

- [ ] **Step 6: Commit**

```bash
git add internal/adapter/postgres/migrate_test.go internal/adapter/postgres/setup_test.go
git commit -m "test(postgres): apply real Goose baseline instead of GORM AutoMigrate"
```

---

## Task 6: Remove all Prisma artifacts

Only after Task 3's gate passed and Task 5 is green.

**Files:** Delete `.prisma`, `prisma.config.ts`, Prisma seeds; Modify `package.json`; regenerate lockfile

- [ ] **Step 1: Delete Prisma schema + config + coupled seeds**

Run:
```bash
rm schemas/app/schema.prisma schemas/reporting/schema.prisma schemas/usage/schema.prisma
rm schemas/app/prisma.config.ts schemas/reporting/prisma.config.ts schemas/usage/prisma.config.ts
rm schemas/app/seed.js schemas/reporting/seed_test.js
```
Keep `schemas/app/seed.sql`.

- [ ] **Step 2: Prune `package.json`**

Edit `package.json`:
- Remove the `prisma:*` scripts (`prisma:generate`, `prisma:format`, `prisma:push`, `prisma:seed`, `prisma:reporting:format`, `prisma:reporting:push`, `prisma:usage:format`, `prisma:usage:push`). Keep `deploy:*`, `tunnel:*`, `ngrok`.
- Remove from `dependencies`: `@prisma/adapter-pg`, `@prisma/client`, `dotenv`, `pg` (only used by the deleted `seed.js`).
- Remove from `devDependencies`: `@faker-js/faker`, `prisma`, `tsx` (only used by the deleted Prisma configs/seeds).
- If `dependencies`/`devDependencies` end up empty, delete the empty blocks.

- [ ] **Step 3: Regenerate the lockfile and prune node_modules**

Run:
```bash
rm -rf node_modules pnpm-lock.yaml
pnpm install
```
Expected: a fresh `pnpm-lock.yaml` with no Prisma packages. Verify:
```bash
grep -i prisma pnpm-lock.yaml || echo "no prisma in lockfile"
```
Expected: `no prisma in lockfile`.

- [ ] **Step 4: Confirm no Prisma references remain in build/config**

Run:
```bash
grep -rin "prisma" Makefile package.json --include="*" || echo "clean"
grep -rin "prisma" internal --include="*.go" | grep -v "_test.go" || echo "go clean"
```
Expected: `clean` / `go clean` (any remaining hits should only be incidental comments, e.g. `report_repo.go` historical notes — review and update if misleading).

- [ ] **Step 5: Commit**

```bash
git add -A schemas package.json pnpm-lock.yaml
git commit -m "chore: remove Prisma schemas, configs, seeds, and dependencies"
```

---

## Task 7: Update documentation & env comments

**Files:** Modify `CLAUDE.md`, `CONTEXT.md`, `README.md`, `.env.example`

- [ ] **Step 1: Update `CLAUDE.md`**

Find the guidance describing Prisma as schema source-of-truth with "no migrations / clean `db push`". Replace with: schema is managed by Goose SQL migrations in `schemas/<db>/migrations/`; the source of truth is the migration history (baseline `00001_baseline.sql` was generated from the former Prisma schemas at the cutover). Apply via `make db-migrate-all`; create new ones via `make db-migrate-create name=...`.

- [ ] **Step 2: Update `CONTEXT.md` and `README.md`**

Replace any `prisma db push` / `make db-push*` instructions with the `make db-migrate*` equivalents and the stamp note below. Add a short "Existing databases" runbook subsection:

> For a database that already has the schema (created by the old `prisma db push`), do not run the baseline. Stamp it as applied instead:
> ```sql
> CREATE TABLE IF NOT EXISTS goose_db_version (
>   id SERIAL PRIMARY KEY,
>   version_id BIGINT NOT NULL,
>   is_applied BOOLEAN NOT NULL,
>   tstamp TIMESTAMP NULL DEFAULT now()
> );
> INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, true), (1, true);
> ```
> Then `make db-migrate-status` shows the baseline as applied and future migrations run normally.

- [ ] **Step 3: Update `.env.example`**

Find comments referencing `pnpm prisma:*:push` / `prisma db push` and rewrite them to reference the corresponding `make db-migrate*` targets. Leave the `DATABASE_URL`/`REPORTING_DATABASE_URL`/`USAGE_DATABASE_URL` variables themselves unchanged.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md CONTEXT.md README.md .env.example
git commit -m "docs: document Goose migration workflow, retire Prisma references"
```

---

## Task 8: Final end-to-end verification

**Files:** none

- [ ] **Step 1: Build + vet**

Run:
```bash
go build ./... && go vet ./...
```
Expected: clean, no errors.

- [ ] **Step 2: Apply all migrations against a fresh local stack**

Run:
```bash
make down && make up
# wait for Postgres healthy, then (with .env loaded):
make db-migrate-all
```
Expected: each of the three `goose up` invocations prints `OK 00001_baseline.sql`.

- [ ] **Step 3: Seed runs**

Run:
```bash
make db-seed
```
Expected: psql applies `schemas/app/seed.sql` with no errors (idempotent — `ON CONFLICT DO NOTHING`).

- [ ] **Step 4: Integration tests pass on the real-migration harness**

Run:
```bash
make test-integration
```
Expected: PASS.

- [ ] **Step 5: Final Prisma-reference sweep**

Run:
```bash
grep -rin "prisma" Makefile package.json CLAUDE.md CONTEXT.md README.md .env.example || echo "docs/build clean"
```
Expected: `docs/build clean` (or only intentional historical mentions).

- [ ] **Step 6: Confirm worktree status**

Run:
```bash
git status --short && git log --oneline origin/main..HEAD
```
Expected: clean tree; commit list shows Tasks 1–7. Ready for review / PR via the finishing-a-development-branch skill.

---

## Self-Review Notes

- **Spec coverage:** baseline generation (T2), zero-drift gate (T3), CLI-only toolchain (T1+T4), real-migration tests (T5), full Prisma removal incl. seeds (T6), docs incl. stamp runbook (T7), DoD verification (T8). All spec sections mapped.
- **Ordering invariants:** Prisma must exist for T2/T3 → both precede removal in T6. Goose lib + baselines precede the test swap in T5. Gate (T3) must pass before T6.
- **Known adaptation point:** T5 Step 2 threads `*testing.T` into the helper because the original AutoMigrate call sits inside a `sync.Once` closure — the executor must read the real `setup_test.go` structure and wire accordingly.
