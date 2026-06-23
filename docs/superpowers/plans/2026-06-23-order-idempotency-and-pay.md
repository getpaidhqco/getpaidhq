# Idempotent CreateOrder (via idempo) + `/orders/{id}/pay` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `CreateOrder` idempotent using the `idempo` HTTP middleware backed by our own Postgres `Store` (our table, both storage adapters), and split PSP session init into a retryable `POST /orders/{id}/pay`.

**Architecture:** `idempo` (pinned dep) owns all idempotency logic — fingerprint, fencing token, single-winner concurrency, response capture/replay, 409/422. We implement only a 3-method `port.IdempotencyStore` (gorm + pgx, conformance-tested) on a new `idempotency_requests` table, plus a thin shim that adapts it to `idempo.Store` and org-scopes the key. The middleware is group-scoped on `/orders` so it runs after authn. Separately, PSP init moves out of `CreateOrder` into `InitOrderPayment` / `POST /orders/{id}/pay`.

**Tech Stack:** Go 1.24, Fuego v0.19, GORM + jackc/pgx/v5, Goose, testcontainers, `github.com/eben-vranken/idempo`.

**Spec:** `docs/superpowers/specs/2026-06-23-order-idempotency-and-pay-design.md`

**Out of scope (separate prerequisite work):** wrapping `CreateOrder`'s writes in `RunInTx` and the gorm `RunInTx` ctx/SAVEPOINT fix. This plan does not touch transaction boundaries.

---

## Phase 1 — Idempotency: `idempo` + our Postgres `Store`

### Task 1: Add the dependency and the `port.IdempotencyStore` contract

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `internal/core/port/repository.go`

- [ ] **Step 1: Add the idempo module**

Run: `go get github.com/eben-vranken/idempo@latest`
Expected: `go.mod` gains a pinned `github.com/eben-vranken/idempo vX.Y.Z` line. (Pin the resolved version; do not use a branch.)

- [ ] **Step 2: Define the port** (append to `internal/core/port/repository.go`, after `IdempotencyKeyRepository`)

```go
// IdempotencyClaimStatus mirrors idempo's claim outcomes. String values are
// identical to idempo.ClaimStatus so the http-layer shim can cast directly.
type IdempotencyClaimStatus string

const (
	IdempotencyNew       IdempotencyClaimStatus = "new"
	IdempotencyPending   IdempotencyClaimStatus = "pending"
	IdempotencyCompleted IdempotencyClaimStatus = "completed"
	IdempotencyConflict  IdempotencyClaimStatus = "conflict"
)

// IdempotencyClaim is the result of a Claim. Code/Headers/Body are populated
// only when Status is IdempotencyCompleted.
type IdempotencyClaim struct {
	Status  IdempotencyClaimStatus
	Code    int
	Headers []byte
	Body    []byte
}

// IdempotencyStore is the persistence behind the idempo middleware. It mirrors
// idempo.Store one-to-one so a thin shim can adapt it without importing idempo
// into core. Contract (per idempo): Claim must make exactly ONE concurrent
// caller win (IdempotencyNew); an expired pending/completed row is reclaimable
// as new. Complete and Abandon MUST be no-ops when the stored token does not
// match, or when the row is not pending (fencing).
type IdempotencyStore interface {
	Claim(ctx context.Context, key, requestHash, token string) (IdempotencyClaim, error)
	Complete(ctx context.Context, key, token string, statusCode int, headers, body []byte) error
	Abandon(ctx context.Context, key, token string) error
}
```

- [ ] **Step 3: Compile**

Run: `go build ./...`
Expected: builds (the port has no implementer yet; that's fine — nothing wires it).

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum internal/core/port/repository.go
git commit -m "feat(port): add IdempotencyStore for idempo-backed request idempotency"
```

---

### Task 2: Goose migration for `idempotency_requests`

**Files:**
- Create: `schemas/app/migrations/00004_idempotency_requests.sql`

- [ ] **Step 1: Write the migration** (match neighbours' style — `TIMESTAMP(3)`, `CURRENT_TIMESTAMP`)

```sql
-- +goose Up
CREATE TABLE "idempotency_requests" (
    "key"              TEXT        NOT NULL,            -- "<orgId>:<clientKey>" (org-scoped by the http shim)
    "request_hash"     TEXT        NOT NULL,            -- idempo's sha256(method+path+body)
    "state"            TEXT        NOT NULL,            -- 'pending' | 'completed'
    "token"            TEXT        NOT NULL,            -- fencing token
    "response_code"    INTEGER,
    "response_headers" BYTEA,
    "response_body"    BYTEA,
    "expires_at"       TIMESTAMP(3) NOT NULL,
    "created_at"       TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at"       TIMESTAMP(3) NOT NULL,

    CONSTRAINT "idempotency_requests_pkey" PRIMARY KEY ("key")
);
CREATE INDEX "idempotency_requests_expires_at" ON "idempotency_requests" ("expires_at");

-- +goose Down
DROP TABLE "idempotency_requests";
```

- [ ] **Step 2: Apply to the local app DB**

Run: `make db-migrate-all`
Expected: `00004_idempotency_requests` applied, no error.

- [ ] **Step 3: Commit**

```bash
git add schemas/app/migrations/00004_idempotency_requests.sql
git commit -m "feat(db): add idempotency_requests table"
```

---

### Task 3: Shared conformance cases for `IdempotencyStore`

This task writes the cross-driver test first (TDD). It won't compile until `RepoSet` gains the field (Task 4) and an adapter implements it (Tasks 5–6). Write it now so both drivers are held to it.

**Files:**
- Modify: `internal/adapter/storage/storagetest/conformance.go`

- [ ] **Step 1: Add the `RepoSet` field** (in the struct around line 34)

```go
	IdempotencyStore port.IdempotencyStore
```

- [ ] **Step 2: Register the sub-test** (in `RunConformance`, after the existing `IdempotencyClaim` line)

```go
	t.Run("IdempotencyStore", func(t *testing.T) { testIdempotencyStore(t, ctx, rs) })
```

- [ ] **Step 3: Write the test function** (append to `conformance.go`)

```go
func testIdempotencyStore(t *testing.T, ctx context.Context, rs RepoSet) {
	key := lib.GenerateId("idemreq")
	hashA, hashB := "hash-a", "hash-b"

	// First claim wins as New.
	c, err := rs.IdempotencyStore.Claim(ctx, key, hashA, "tok-1")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyNew, c.Status)

	// Concurrent claim while pending → Pending (handler must not run twice).
	c, err = rs.IdempotencyStore.Claim(ctx, key, hashA, "tok-2")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyPending, c.Status)

	// Fencing: Complete with the WRONG token is a no-op (row stays pending).
	require.NoError(t, rs.IdempotencyStore.Complete(ctx, key, "tok-WRONG", 200, []byte(`{"h":1}`), []byte(`{"order":"x"}`)))
	c, err = rs.IdempotencyStore.Claim(ctx, key, hashA, "tok-3")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyPending, c.Status, "wrong-token Complete must not complete the row")

	// Complete with the owning token → stores the response.
	require.NoError(t, rs.IdempotencyStore.Complete(ctx, key, "tok-1", 201, []byte(`{"h":1}`), []byte(`{"order":"x"}`)))

	// Replay: same key + same hash → Completed with the exact stored response.
	c, err = rs.IdempotencyStore.Claim(ctx, key, hashA, "tok-4")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyCompleted, c.Status)
	assert.Equal(t, 201, c.Code)
	assert.Equal(t, []byte(`{"h":1}`), c.Headers)
	assert.Equal(t, []byte(`{"order":"x"}`), c.Body)

	// Same key, DIFFERENT hash → Conflict.
	c, err = rs.IdempotencyStore.Claim(ctx, key, hashB, "tok-5")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyConflict, c.Status)

	// Abandon releases a fresh pending claim so the key is retryable.
	key2 := lib.GenerateId("idemreq")
	c, err = rs.IdempotencyStore.Claim(ctx, key2, hashA, "tok-a")
	require.NoError(t, err)
	require.Equal(t, port.IdempotencyNew, c.Status)
	require.NoError(t, rs.IdempotencyStore.Abandon(ctx, key2, "tok-WRONG")) // no-op (fenced)
	c, err = rs.IdempotencyStore.Claim(ctx, key2, hashA, "tok-b")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyPending, c.Status, "wrong-token Abandon must not release")
	require.NoError(t, rs.IdempotencyStore.Abandon(ctx, key2, "tok-a")) // real release
	c, err = rs.IdempotencyStore.Claim(ctx, key2, hashA, "tok-c")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyNew, c.Status, "claim after release wins again")

	// Single-winner under concurrency.
	key3 := lib.GenerateId("idemreq")
	const n = 12
	results := make([]port.IdempotencyClaimStatus, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cc, e := rs.IdempotencyStore.Claim(ctx, key3, hashA, fmt.Sprintf("tok-%d", i))
			require.NoError(t, e)
			results[i] = cc.Status
		}(i)
	}
	wg.Wait()
	newCount := 0
	for _, s := range results {
		if s == port.IdempotencyNew {
			newCount++
		}
	}
	assert.Equal(t, 1, newCount, "exactly one concurrent claim wins New")
}
```

- [ ] **Step 4: Add the `sync` import** to `conformance.go` if not present.

- [ ] **Step 5: Commit** (compiles after Task 4–6; commit alongside Task 4 if your reviewer prefers green commits — otherwise stage here and commit with Task 4)

```bash
git add internal/adapter/storage/storagetest/conformance.go
git commit -m "test(storagetest): conformance cases for IdempotencyStore"
```

---

### Task 4: GORM `IdempotencyStore` adapter

**Files:**
- Create: `internal/adapter/storage/postgresgorm/idempotency_store.go`
- Modify: `internal/adapter/storage/postgresgorm/conformance_test.go`

- [ ] **Step 1: Implement the store**

```go
package postgresgorm

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"getpaidhq/internal/core/port"
)

// IdempotencyRequestEntity maps the idempotency_requests table (the idempo
// Store backing). updated_at is NOT NULL with no DB default, so it is set on
// every write; created_at has a DB default and is omitted.
type IdempotencyRequestEntity struct {
	Key             string    `gorm:"column:key;primaryKey"`
	RequestHash     string    `gorm:"column:request_hash"`
	State           string    `gorm:"column:state"`
	Token           string    `gorm:"column:token"`
	ResponseCode    *int      `gorm:"column:response_code"`
	ResponseHeaders []byte    `gorm:"column:response_headers"`
	ResponseBody    []byte    `gorm:"column:response_body"`
	ExpiresAt       time.Time `gorm:"column:expires_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (IdempotencyRequestEntity) TableName() string { return "idempotency_requests" }

type IdempotencyStore struct {
	db           *gorm.DB
	lockTTL      time.Duration
	retentionTTL time.Duration
}

func NewIdempotencyStore(db *gorm.DB, lockTTL, retentionTTL time.Duration) port.IdempotencyStore {
	return &IdempotencyStore{db: db, lockTTL: lockTTL, retentionTTL: retentionTTL}
}

var _ port.IdempotencyStore = (*IdempotencyStore)(nil)

// Claim: sweep an expired row for this key, then INSERT ... ON CONFLICT DO
// NOTHING. RowsAffected==1 means we won (New). On conflict we read the holder
// and report Pending / Completed (same hash → replay) / Conflict (hash differs).
// The conditional insert is the single-winner arbiter; the sweep only enables
// reclaiming an expired row (mirrors the existing IdempotencyKeyRepo pattern).
func (r *IdempotencyStore) Claim(ctx context.Context, key, requestHash, token string) (port.IdempotencyClaim, error) {
	now := time.Now().UTC()

	if err := dbFromCtx(ctx, r.db).
		Where("key = ? AND expires_at <= ?", key, now).
		Delete(&IdempotencyRequestEntity{}).Error; err != nil {
		return port.IdempotencyClaim{}, err
	}

	entity := IdempotencyRequestEntity{
		Key:         key,
		RequestHash: requestHash,
		State:       string(port.IdempotencyPending),
		Token:       token,
		ExpiresAt:   now.Add(r.lockTTL),
	}
	res := dbFromCtx(ctx, r.db).Clauses(clause.OnConflict{DoNothing: true}).Create(&entity)
	if res.Error != nil {
		return port.IdempotencyClaim{}, res.Error
	}
	if res.RowsAffected == 1 {
		return port.IdempotencyClaim{Status: port.IdempotencyNew}, nil
	}

	var existing IdempotencyRequestEntity
	if err := dbFromCtx(ctx, r.db).Where("key = ?", key).First(&existing).Error; err != nil {
		return port.IdempotencyClaim{}, err
	}
	switch existing.State {
	case string(port.IdempotencyPending):
		return port.IdempotencyClaim{Status: port.IdempotencyPending}, nil
	case string(port.IdempotencyCompleted):
		if existing.RequestHash != requestHash {
			return port.IdempotencyClaim{Status: port.IdempotencyConflict}, nil
		}
		code := 0
		if existing.ResponseCode != nil {
			code = *existing.ResponseCode
		}
		return port.IdempotencyClaim{
			Status:  port.IdempotencyCompleted,
			Code:    code,
			Headers: existing.ResponseHeaders,
			Body:    existing.ResponseBody,
		}, nil
	default:
		return port.IdempotencyClaim{Status: port.IdempotencyConflict}, nil
	}
}

// Complete is token-fenced and pending-only: a stale request can't overwrite a
// claim a newer request now owns. No matching row → no-op (RowsAffected 0).
func (r *IdempotencyStore) Complete(ctx context.Context, key, token string, statusCode int, headers, body []byte) error {
	now := time.Now().UTC()
	return dbFromCtx(ctx, r.db).
		Model(&IdempotencyRequestEntity{}).
		Where("key = ? AND token = ? AND state = ?", key, token, string(port.IdempotencyPending)).
		Updates(map[string]any{
			"state":            string(port.IdempotencyCompleted),
			"response_code":    statusCode,
			"response_headers": headers,
			"response_body":    body,
			"expires_at":       now.Add(r.retentionTTL),
			"updated_at":       now,
		}).Error
}

// Abandon is token-fenced and pending-only.
func (r *IdempotencyStore) Abandon(ctx context.Context, key, token string) error {
	return dbFromCtx(ctx, r.db).
		Where("key = ? AND token = ? AND state = ?", key, token, string(port.IdempotencyPending)).
		Delete(&IdempotencyRequestEntity{}).Error
}
```

- [ ] **Step 2: Add to the gorm conformance factory** (`conformance_test.go`, inside the `RepoSet{...}`)

```go
		IdempotencyStore: NewIdempotencyStore(db, time.Minute, 24*time.Hour),
```
(add `"time"` to that file's imports)

- [ ] **Step 3: Run the gorm conformance suite**

Run: `go test -tags integration ./internal/adapter/storage/postgresgorm/ -run TestConformance/IdempotencyStore -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/storage/postgresgorm/idempotency_store.go internal/adapter/storage/postgresgorm/conformance_test.go internal/adapter/storage/storagetest/conformance.go
git commit -m "feat(postgresgorm): IdempotencyStore implementing idempo's Store contract"
```

---

### Task 5: pgx `IdempotencyStore` adapter (parity)

**Files:**
- Create: `internal/adapter/storage/postgrespgx/idempotency_store.go`
- Modify: `internal/adapter/storage/postgrespgx/conformance_test.go`

- [ ] **Step 1: Implement the store** (hand-written SQL; same behaviour as gorm)

```go
package postgrespgx

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/port"
)

type IdempotencyStore struct {
	pool         *pgxpool.Pool
	lockTTL      time.Duration
	retentionTTL time.Duration
}

func NewIdempotencyStore(pool *pgxpool.Pool, lockTTL, retentionTTL time.Duration) port.IdempotencyStore {
	return &IdempotencyStore{pool: pool, lockTTL: lockTTL, retentionTTL: retentionTTL}
}

var _ port.IdempotencyStore = (*IdempotencyStore)(nil)

func (r *IdempotencyStore) Claim(ctx context.Context, key, requestHash, token string) (port.IdempotencyClaim, error) {
	now := time.Now().UTC()
	db := dbFromCtx(ctx, r.pool)

	if _, err := db.Exec(ctx,
		`DELETE FROM idempotency_requests WHERE key = $1 AND expires_at <= $2`, key, now); err != nil {
		return port.IdempotencyClaim{}, err
	}

	tag, err := db.Exec(ctx, `
		INSERT INTO idempotency_requests (key, request_hash, state, token, expires_at, updated_at)
		VALUES ($1, $2, 'pending', $3, $4, $5)
		ON CONFLICT (key) DO NOTHING`,
		key, requestHash, token, now.Add(r.lockTTL), now)
	if err != nil {
		return port.IdempotencyClaim{}, err
	}
	if tag.RowsAffected() == 1 {
		return port.IdempotencyClaim{Status: port.IdempotencyNew}, nil
	}

	var (
		state    string
		hash     string
		code     *int
		headers  []byte
		body     []byte
	)
	row := db.QueryRow(ctx,
		`SELECT state, request_hash, response_code, response_headers, response_body
		   FROM idempotency_requests WHERE key = $1`, key)
	if err := row.Scan(&state, &hash, &code, &headers, &body); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Holder vanished between the conflict and the read (TTL sweep race);
			// safest is to report Pending so the caller does not double-run.
			return port.IdempotencyClaim{Status: port.IdempotencyPending}, nil
		}
		return port.IdempotencyClaim{}, err
	}

	switch port.IdempotencyClaimStatus(state) {
	case port.IdempotencyPending:
		return port.IdempotencyClaim{Status: port.IdempotencyPending}, nil
	case port.IdempotencyCompleted:
		if hash != requestHash {
			return port.IdempotencyClaim{Status: port.IdempotencyConflict}, nil
		}
		c := 0
		if code != nil {
			c = *code
		}
		return port.IdempotencyClaim{Status: port.IdempotencyCompleted, Code: c, Headers: headers, Body: body}, nil
	default:
		return port.IdempotencyClaim{Status: port.IdempotencyConflict}, nil
	}
}

func (r *IdempotencyStore) Complete(ctx context.Context, key, token string, statusCode int, headers, body []byte) error {
	now := time.Now().UTC()
	_, err := dbFromCtx(ctx, r.pool).Exec(ctx, `
		UPDATE idempotency_requests
		   SET state = 'completed', response_code = $3, response_headers = $4,
		       response_body = $5, expires_at = $6, updated_at = $7
		 WHERE key = $1 AND token = $2 AND state = 'pending'`,
		key, token, statusCode, headers, body, now.Add(r.retentionTTL), now)
	return err
}

func (r *IdempotencyStore) Abandon(ctx context.Context, key, token string) error {
	_, err := dbFromCtx(ctx, r.pool).Exec(ctx,
		`DELETE FROM idempotency_requests WHERE key = $1 AND token = $2 AND state = 'pending'`, key, token)
	return err
}
```

- [ ] **Step 2: Add to the pgx conformance factory** (`postgrespgx/conformance_test.go`, inside its `RepoSet{...}`)

```go
		IdempotencyStore: NewIdempotencyStore(pool, time.Minute, 24*time.Hour),
```
(add `"time"` import if needed)

- [ ] **Step 3: Run BOTH suites (parity)**

Run: `go test -tags integration ./internal/adapter/storage/... -run TestConformance/IdempotencyStore -v`
Expected: PASS for both `postgresgorm` and `postgrespgx`.

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/storage/postgrespgx/idempotency_store.go internal/adapter/storage/postgrespgx/conformance_test.go
git commit -m "feat(postgrespgx): IdempotencyStore at parity with gorm"
```

---

### Task 6: Expose the store in app wiring (`repoSet`)

**Files:**
- Modify: `internal/config/repos.go`

- [ ] **Step 1: Add the field** (to `type repoSet struct`, near `idempotency`)

```go
	idempotencyStore    port.IdempotencyStore
```

- [ ] **Step 2: Build it in the gorm branch** (`newGormRepoSet`, near the `idempotency:` line)

```go
		idempotencyStore:    postgresgorm.NewIdempotencyStore(db, env.IdempotencyLockTTL, env.IdempotencyRetentionTTL),
```

- [ ] **Step 3: Build it in the pgx branch** (`newPgxRepoSet`)

```go
		idempotencyStore:    postgrespgx.NewIdempotencyStore(pool, env.IdempotencyLockTTL, env.IdempotencyRetentionTTL),
```

(The `env.Idempotency*` fields land in Task 7. If implementing strictly in order, temporarily pass `time.Minute, 24*time.Hour` literals and replace in Task 7.)

- [ ] **Step 4: Compile**

Run: `go build ./...`
Expected: builds (once Task 7 env fields exist, or with literals).

- [ ] **Step 5: Commit**

```bash
git add internal/config/repos.go
git commit -m "feat(config): build IdempotencyStore in both repo sets"
```

---

### Task 7: Env config for the two TTLs

**Files:**
- Modify: `internal/lib/env.go`
- Modify: `.env.example`

- [ ] **Step 1: Add struct fields** (to `type Env struct`, near `HatchetBillingSweepInterval`)

```go
	IdempotencyLockTTL      time.Duration `mapstructure:"IDEMPOTENCY_LOCK_TTL"`
	IdempotencyRetentionTTL time.Duration `mapstructure:"IDEMPOTENCY_RETENTION_TTL"`
```

- [ ] **Step 2: Defaults + binds + assignment** (in `NewEnv()`, mirroring `HATCHET_BILLING_SWEEP_INTERVAL`)

```go
	viper.SetDefault("IDEMPOTENCY_LOCK_TTL", "1m")
	viper.SetDefault("IDEMPOTENCY_RETENTION_TTL", "24h")
	// ...
	viper.BindEnv("IDEMPOTENCY_LOCK_TTL")
	viper.BindEnv("IDEMPOTENCY_RETENTION_TTL")
	// ... in the explicit-assignment block:
	env.IdempotencyLockTTL = viper.GetDuration("IDEMPOTENCY_LOCK_TTL")
	env.IdempotencyRetentionTTL = viper.GetDuration("IDEMPOTENCY_RETENTION_TTL")
```

- [ ] **Step 3: Document in `.env.example`**

```
# Idempotency (idempo) — how long an in-flight claim is held, and how long a
# completed response stays replayable.
IDEMPOTENCY_LOCK_TTL=1m
IDEMPOTENCY_RETENTION_TTL=24h
```

- [ ] **Step 4: Build + commit**

Run: `go build ./...`
```bash
git add internal/lib/env.go .env.example
git commit -m "feat(config): IDEMPOTENCY_LOCK_TTL / IDEMPOTENCY_RETENTION_TTL"
```

---

### Task 8: The `idempo.Store` shim + org scoping

**Files:**
- Create: `internal/adapter/http/middleware/idempotency.go`

- [ ] **Step 1: Implement the shim**

```go
package middleware

import (
	"context"

	"github.com/eben-vranken/idempo"

	"getpaidhq/internal/core/port"
)

// idempoStore adapts our port.IdempotencyStore to idempo.Store and scopes every
// key by the authenticated org, so two orgs that send the same Idempotency-Key
// can never collide. The order group's middleware runs AFTER authn, so AuthUser
// is on ctx here.
type idempoStore struct{ store port.IdempotencyStore }

func (a idempoStore) Claim(ctx context.Context, key, requestHash, token string) (idempo.ClaimResult, error) {
	c, err := a.store.Claim(ctx, scopeKey(ctx, key), requestHash, token)
	return idempo.ClaimResult{
		Status:  idempo.ClaimStatus(c.Status),
		Code:    c.Code,
		Headers: c.Headers,
		Body:    c.Body,
	}, err
}

func (a idempoStore) Complete(ctx context.Context, key, token string, statusCode int, headers, body []byte) error {
	return a.store.Complete(ctx, scopeKey(ctx, key), token, statusCode, headers, body)
}

func (a idempoStore) Abandon(ctx context.Context, key, token string) error {
	return a.store.Abandon(ctx, scopeKey(ctx, key), token)
}

func scopeKey(ctx context.Context, key string) string {
	if u, ok := AuthUserFrom(ctx); ok {
		return u.OrgId + ":" + key
	}
	return ":" + key
}

// NewIdempotencyMiddleware builds the idempo middleware over our store.
func NewIdempotencyMiddleware(store port.IdempotencyStore, opts idempo.Options) func(http.Handler) http.Handler {
	return idempo.New(idempoStore{store: store}, opts).Handler
}
```
(add the `net/http` import alongside the others)

- [ ] **Step 2: Build + commit**

Run: `go build ./...`
```bash
git add internal/adapter/http/middleware/idempotency.go
git commit -m "feat(http): idempo Store shim with per-org key scoping"
```

---

### Task 9: Mount the middleware on the order group

**Files:**
- Modify: `internal/adapter/http/order_handler.go`
- Modify: `internal/config/app.go`

- [ ] **Step 1: Give `OrderHandler` the middleware** (struct + constructor)

```go
type OrderHandler struct {
	service *service.OrderService
	logger  port.Logger
	authz   port.Authz
	idem    func(http.Handler) http.Handler
}

func NewOrderHandler(orderService *service.OrderService, logger port.Logger, authz port.Authz, idem func(http.Handler) http.Handler) *OrderHandler {
	return &OrderHandler{service: orderService, logger: logger, authz: authz, idem: idem}
}
```
(add `net/http` import)

- [ ] **Step 2: Attach it to the group** (in `RegisterRoutes`)

```go
	g := fuego.Group(s, "/orders",
		option.Tags("Orders"),
		option.Middleware(o.idem), // idempo; no-ops when no Idempotency-Key header
	)
```

- [ ] **Step 3: Wire it in `app.go`** (build the middleware, pass to the handler)

```go
	idemMW := middleware.NewIdempotencyMiddleware(repos.idempotencyStore, idempo.Options{Logger: slogLogger})
	// ...
	Order: handler.NewOrderHandler(orderService, logger, authzEngine, idemMW),
```
Use the app's `*slog.Logger` for `idempo.Options.Logger` (see `internal/lib/logger.go`; if no accessor exists, pass `slog.Default()`). Import `github.com/eben-vranken/idempo` and the `middleware` package in `app.go`.

- [ ] **Step 4: Build + run unit tests**

Run: `go build ./... && make test`
Expected: builds; existing tests pass (handler constructor callers updated).

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/http/order_handler.go internal/config/app.go
git commit -m "feat(http): mount idempo middleware on the /orders group"
```

---

### Task 10: HTTP-level idempotency test

**Files:**
- Modify/Create: `internal/adapter/http/order_handler_test.go` (use the existing httptest harness)

- [ ] **Step 1: Write the tests**

Behaviours (use the real authn+authz httptest harness already used by order/http tests; build the harness with the idempo middleware backed by a store against the test DB or an in-memory `port.IdempotencyStore` fake — a fake is fine here since the Store itself is covered by conformance):

```go
// 1. Same Idempotency-Key + same body twice → ONE order; second response carries
//    header "Idempotency-Replayed: true" and identical body.
// 2. Same key + DIFFERENT body → 422 on the second call.
// 3. No Idempotency-Key header → each call creates a new order (today's behaviour).
```

- [ ] **Step 2: Run**

Run: `go test ./internal/adapter/http/ -run Idempoten -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/adapter/http/order_handler_test.go
git commit -m "test(http): CreateOrder idempotency via Idempotency-Key"
```

---

## Phase 2 — Split PSP init into `POST /orders/{id}/pay`

### Task 11: `orders.payment_session` column + domain + row mapping

**Files:**
- Create: `schemas/app/migrations/00005_orders_payment_session.sql`
- Modify: `internal/core/domain/order.go`
- Modify: `internal/adapter/storage/postgresgorm/order_row.go`, `internal/adapter/storage/postgrespgx/order_row.go` (filenames per existing order mapping)

- [ ] **Step 1: Migration**

```sql
-- +goose Up
ALTER TABLE "orders" ADD COLUMN "payment_session" JSONB;

-- +goose Down
ALTER TABLE "orders" DROP COLUMN "payment_session";
```
Run: `make db-migrate-all`

- [ ] **Step 2: Domain field** — add `PaymentSession any` to `domain.Order` (JSON-encoded `InitPaymentResponse`).

- [ ] **Step 3: Map the column in both adapters' order row** (nullable JSONB ↔ `any`, following the existing metadata/jsonb handling in each `order_row`). Extend the conformance `testCartOrderItem` (or add a focused case) to round-trip `PaymentSession`.

- [ ] **Step 4: Run conformance + commit**

Run: `go test -tags integration ./internal/adapter/storage/... -run TestConformance -v`
```bash
git add schemas/app/migrations/00005_orders_payment_session.sql internal/core/domain/order.go internal/adapter/storage/postgres*/order_row.go internal/adapter/storage/storagetest/conformance.go
git commit -m "feat(orders): persist payment_session on the order (both drivers)"
```

---

### Task 12: `InitOrderPayment` service + remove PSP from `CreateOrder`

**Files:**
- Modify: `internal/core/service/order.go`
- Modify: `internal/core/domain/order.go` (drop `Psp` from `CreateOrderResponse`)

- [ ] **Step 1: Remove the PSP block from `CreateOrder`** (delete `order.go:298-318`'s gateway/InitPayment block and the `Psp` field from the returned `CreateOrderResponse`; keep the post-write `FindById` + return `{Order}` only). `CreateOrder` makes no gateway call.

- [ ] **Step 2: Add `InitOrderPayment`**

```go
// InitOrderPayment initialises (or returns) the PSP payment session for an
// existing pending order. Idempotent on the stored session: a second call (or a
// retry after a gateway failure) returns the same session, never a duplicate.
func (s *OrderService) InitOrderPayment(ctx context.Context, orgId, orderId string, opts map[string]any) (domain.InitPaymentResponse, error) {
	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		return domain.InitPaymentResponse{}, err
	}
	if order.Status != domain.OrderStatusPending { // use the real pending constant
		return domain.InitPaymentResponse{}, lib.NewCustomError(lib.ConflictError, "order is not payable", nil)
	}
	if order.PaymentSession != nil {
		return domain.InitPaymentResponse{PspResponse: order.PaymentSession}, nil
	}
	cart, err := s.cartRepository.FindById(ctx, orgId, order.CartId)
	if err != nil {
		return domain.InitPaymentResponse{}, err
	}
	customer, err := s.customerRepository.FindById(ctx, orgId, order.CustomerId)
	if err != nil {
		return domain.InitPaymentResponse{}, err
	}
	gw, err := s.gatewayFactory.NewGateway(ctx, orgId, /* order's PSP */)
	if err != nil {
		return domain.InitPaymentResponse{}, err
	}
	resp, err := gw.InitPayment(ctx, domain.InitPaymentCommand{OrgId: orgId, Cart: cart, Order: order, Customer: customer, Options: opts})
	if err != nil {
		return domain.InitPaymentResponse{}, err
	}
	if err := s.orderRepository.SetPaymentSession(ctx, orgId, orderId, resp.PspResponse); err != nil {
		return domain.InitPaymentResponse{}, err
	}
	return resp, nil
}
```
Add `OrderRepository.SetPaymentSession(ctx, orgId, id string, session any) error` to the port and both adapters (a single-column UPDATE). The order must persist which PSP it used (it already routes through `PspId` at creation) — read it from the stored order; if not currently persisted, add it to the order row in Task 11.

- [ ] **Step 3: Unit tests** — `InitOrderPayment`: inits once; returns the stored session on a second call (gateway called once via a stub); errors on a non-pending order.

Run: `make test`
- [ ] **Step 4: Commit**

```bash
git add internal/core/service/order.go internal/core/domain/order.go internal/core/port/repository.go internal/adapter/storage/postgres*/*
git commit -m "feat(orders): InitOrderPayment; CreateOrder no longer calls the PSP"
```

---

### Task 13: `POST /orders/{id}/pay` handler + route + Cedar + response change

**Files:**
- Modify: `internal/adapter/http/order_handler.go`
- Modify: `policy.cedar`
- Modify: `internal/core/port/*` (Cedar action constant, e.g. `ActionPayOrder`)

- [ ] **Step 1: Drop `Psp` from `CreateOrderResponse`** (handler struct at `order_handler.go:39`) and stop setting it at `:89-92` (return `{Order}` only). Update any test asserting `resp.Psp`.

- [ ] **Step 2: Add the handler**

```go
func (o *OrderHandler) Pay(c fuego.ContextWithBody[PayOrderRequest]) (PayOrderResponse, error) {
	authUser := AuthUserFrom(c)
	if !o.authz.Enforce(authUser, port.ActionPayOrder, "") {
		return PayOrderResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	id := c.PathParam("id")
	input, _ := c.Body() // optional gateway options
	resp, err := o.service.InitOrderPayment(c.Context(), authUser.OrgId, id, input.Options)
	if err != nil {
		return PayOrderResponse{}, NewApiErrorFromError(err)
	}
	return PayOrderResponse{Psp: resp.PspResponse}, nil
}
```
Define `PayOrderRequest{ Options map[string]any }` and `PayOrderResponse{ Psp any json:"psp" }`.

- [ ] **Step 3: Register the route** (in `RegisterRoutes`, on `g`)

```go
	fuego.Post(g, "/{id}/pay", o.Pay, option.Summary("Initialise an order's payment session"), option.OperationID("payOrder"))
```

- [ ] **Step 4: Cedar** — add `ActionPayOrder` and a policy line permitting it for the order roles (mirror `ActionCreateOrder`). Add a `port.ActionPayOrder` constant.

- [ ] **Step 5: Tests** — `POST /orders/{id}/pay` returns a session; a second call returns the same session; a non-pending order → 409/conflict; unauthorised → 403.

Run: `make test`
- [ ] **Step 6: Commit**

```bash
git add internal/adapter/http/order_handler.go policy.cedar internal/core/port/*
git commit -m "feat(http): POST /orders/{id}/pay; CreateOrderResponse drops psp"
```

---

### Task 14: End-to-end integration + full suite

**Files:**
- Modify/Create: an order e2e in the integration suite

- [ ] **Step 1: e2e** — create order (no PSP in response) → `POST /{id}/pay` → session; retry `/pay` → identical session; replay `POST /orders` with the same `Idempotency-Key` → same order, no second subscription.

- [ ] **Step 2: Full CI gate**

Run: `make ci` then `make test-integration`
Expected: green across all packages (both drivers).

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "test(e2e): order create (idempotent) + pay split"
```

---

## Self-review notes

- **Spec coverage:** §2 idempo+Store → Tasks 1–10; §3 `/pay` → Tasks 11–13; §3.1 response change → Tasks 12–13; §4 data model → Tasks 2, 11; §6 placement → Tasks 6–9; §7 transactions → no code (Store calls are autocommit by construction; nothing wraps them); §8 testing → Tasks 3, 10, 12–14.
- **Parity:** every storage change lands in both `postgresgorm` and `postgrespgx` and is gated by the shared `storagetest` conformance suite (Tasks 3–5, 11).
- **Type consistency:** `port.IdempotencyStore` / `IdempotencyClaim` / `IdempotencyClaimStatus` defined in Task 1 are used verbatim in Tasks 3–6, 8. The shim casts `port.IdempotencyClaimStatus` → `idempo.ClaimStatus` (identical string values).
- **Deliberately deferred:** expiry-by-wall-clock reclaim is exercised implicitly (the sweep runs on every `Claim`); a dedicated short-TTL timing test can be added per-adapter if desired, but the conflict/replay/fencing/single-winner behaviours are the load-bearing ones and are in conformance.
- **Engine parity:** no workflow-engine code is touched.
