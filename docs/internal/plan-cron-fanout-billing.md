# Cron + Per-Org Fan-Out Billing — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the immortal per-subscription `subscription-runner` durable loop on Hatchet with a Lago-style **hourly cron → per-org fan-out → one fresh `billing-cycle-runner` task per due subscription**, so each renewal is a short, *completing* durable task that ages out of the creation-time-partitioned retention model cleanly.

**Architecture:** Three new Hatchet workflows form a fan-out tree, each level doing no work of the next: (1) `billing-sweep` — a cron-triggered standalone task (`WithCron("10 * * * *")`) that lists org ids and spawns one `org-billing` run per org; (2) `org-billing` — a standalone task that queries *that org's* due subscriptions and spawns one `billing-cycle-runner` per due sub (idempotent via `billing_<org>_<sub>_<cycle>`); (3) `billing-cycle-runner` — a **bounded** durable task that runs exactly one cycle (spawn the existing `billing-cycle` charge DAG → optionally wait ≤1h for the PSP webhook → `HandleSubscriptionChargeSuccess/Failure`) and **exits**. State lives on the subscription row (`RenewsAt`/`NextRetryAt`/`CyclesProcessed`/`Status`), so the next hourly sweep picks up the next cycle and paused/cancelled subs simply aren't selected. The old `subscription-runner` is retired and Hatchet's `StartSubscriptionWorkflow` becomes a no-op.

**Tech Stack:** Go 1.24, Hatchet Go SDK (`github.com/hatchet-dev/hatchet/sdks/go` @ v0.86.5), GORM/Postgres, Testcontainers integration tests (`//go:build integration`, `testDB(t)`).

**Why this shape (read first):** see [durable-runner-timeouts.md](durable-runner-timeouts.md) and [subscriptions-on-hatchet.md](subscriptions-on-hatchet.md). The decisive constraint: Hatchet's durable event log is `PARTITION BY RANGE(durable_task_inserted_at)` and retention drops whole partitions by the task's *birth date*, liveness-blind — so an immortal loop has its state deleted out from under it. A per-renewal task completes and ages out safely; that is the entire point of this migration.

---

## Prerequisites & invariants (do not skip)

These hold the design together. Verify before/while implementing:

1. **`HandleSubscriptionChargeSuccess` / `HandleSubscriptionChargeFailure` must be idempotent per cycle.** `billing-cycle-runner` is durable: on eviction/worker-restart it **replays from the top**, and the charge DAG can also be retried. Re-applying the same cycle's result must not double-advance `CyclesProcessed`, double-count `TotalRevenue`, or open duplicate dunning campaigns. If they are not already idempotent, guard them (e.g. no-op if `CyclesProcessed` already past this cycle, or key dunning campaign creation by `(orgId, subId, cycle)`). **Audit this in Task 0.**
2. **The charge itself is already idempotent** via the `billing-cycle` run key `billing_<org>_<sub>_<cycle>` (`cycle = CyclesProcessed`). Keep that key.
3. **BOTH engines must honor the per-tenant reminder config** (engine-parity rule — see CLAUDE.md "Workflow engine"). Most of this plan is engine-agnostic: the new repo queries (`ListIds`, `FindDueForBilling`, `FindUpcomingRenewals`), the reminder stack (`domain.ReminderConfig`, `ReminderConfigService`, `port.ReminderConfigResolver`, `SettingRepository.Upsert`, the HTTP handler) and the Task 0 idempotency guards live in `core/`+`adapter/postgres`+`adapter/http` and are shared. The **billing-trigger** orchestration is intentionally per-engine (Hatchet cron-fan-out vs Temporal durable runner — the one documented divergence). **Reminders are unified across both engines in this plan:** Hatchet via the sweep (Task 5), Temporal via the runner re-resolving config each cycle (Task 9). The **Engine scope & Temporal** section below specifies the consistency model.
4. **Renewal reminders ARE in scope, and are PER-TENANT settings — not an env var** (Tasks 2B, 4B–4E, and the reminder branch in Task 5). The policy (`enabled` + `offsets`) is stored per org in the `settings` table, resolved with a `DefaultReminderConfig()` fallback (mirroring `DunningService.ResolveConfig`), and set by merchants via `PUT /api/billing/reminder-config`. Each `(cycle, offset)` stage sends exactly once via run-key dedup. The old "1 minute before charge" runner spawn is replaced, not preserved literally.
5. **Out of scope (follow-ups, note but don't build here):** the `dunning-runner` eviction fix (separate change — it's bounded so retention-safe, it just needs an eviction policy); **per-plan / per-customer-segment** reminder scoping (this pass is per-org; the resolver is the natural extension point, like `DunningConfigScope`); multi-currency/anniversary date math (Lago-grade) beyond the existing `GetNextChargeDate`.

---

## Engine scope & Temporal

**Engine-parity rule (load-bearing — now also in CLAUDE.md):** every workflow/billing/dunning/reminder change must work on **both** adapters. This plan honors it: the reminder config is shared and respected by both engines.

The two engines differ on exactly **one** axis — the billing *trigger* — because their durability models differ. Everything else, including the per-tenant reminder policy, is identical.

| | Hatchet (after this plan) | Temporal (after this plan, Task 9) |
| --- | --- | --- |
| Billing trigger | hourly cron `billing-sweep` → per-org → fresh `billing-cycle-runner` | durable per-sub `SubscriptionWorkflow` that sleeps the interval, loops, `ContinueAsNew` for history |
| Why different | Hatchet GC's durable logs by birth-date partition → immortal task gets reaped | Temporal timer/history + `ContinueAsNew` → immortal runner is **valid** |
| `StartSubscriptionWorkflow` | **no-op** (cron drives billing) | still starts the per-sub runner |
| Reminders | per-tenant `ReminderConfig`, resolved by the sweep **each tick** (≤1h to apply changes) | per-tenant `ReminderConfig`, resolved by the runner **once per billing cycle** (changes apply next cycle) |

Why the calling code doesn't change: the `Engine` port is satisfied by both adapters, and `order.go` / `subscription_orchestration.go` just call `StartSubscriptionWorkflow` — Hatchet no-ops it (cron takes over), Temporal starts its runner.

### Reminder consistency model (running subscriptions vs. config updates)

A long-running subscription can't be live-patched when a tenant edits the reminder schedule — so:

- **Hatchet** re-resolves the config on every sweep tick (the sweep is stateless), so edits take effect within ≤1 hour for any not-yet-fired stage of the current cycle.
- **Temporal** re-resolves the config **at the top of each billing cycle** (via the `ResolveReminderConfig` activity), commits that cycle's reminder timers, and **ignores further edits until the next cycle**. This is the agreed model: "calculate each loop and set." Live mid-cycle reconfiguration of a running workflow is explicitly out of scope (it would need a signal to the running runner; not worth the complexity).

Both read the **same** `ReminderConfigService` / settings — they differ only in *when* they sample it. Temporal resolves via an **activity** (not a direct repo call) because workflow code must stay deterministic.

## File Structure

| File | Responsibility | Change |
| --- | --- | --- |
| `internal/core/port/repository.go` | repo interfaces | **Modify**: add `OrgRepository.ListIds`, `SubscriptionRepository.FindDueForBilling`, `SubscriptionRepository.FindUpcomingRenewals` |
| `internal/adapter/postgres/org_repo.go` | org persistence | **Modify**: implement `ListIds` |
| `internal/adapter/postgres/subscription_repo.go` | subscription persistence | **Modify**: implement `FindDueForBilling`, `FindUpcomingRenewals` |
| `internal/adapter/postgres/org_repo_test.go` | org repo tests | **Create** |
| `internal/adapter/postgres/subscription_repo_test.go` | sub repo tests | **Create** (or extend if present) |
| `internal/core/domain/reminder_config.go` | `ReminderConfig` value type + parse/marshal + default + setting keys | **Create** |
| `internal/core/port/repository.go` | `SettingRepository` | **Modify**: add `Upsert` |
| `internal/adapter/postgres/setting_repo.go` | settings persistence | **Modify**: implement `Upsert` |
| `internal/core/port/service.go` | service ports | **Modify**: add `ReminderConfigResolver` |
| `internal/core/service/reminder_config.go` | `ReminderConfigService` (per-tenant resolve/set) | **Create** |
| `internal/adapter/http/reminder_config_handler.go` | tenant GET/PUT reminder config | **Create** |
| `internal/config/server.go` | route registration | **Modify**: register reminder-config handler |
| `internal/adapter/hatchet/workflows/types.go` | workflow input structs | **Modify**: add `OrgBillingInput` |
| `internal/adapter/hatchet/workflows/keys.go` | run/event keys | **Modify**: add `OrgBillingRunKey`, `ReminderStageRunKey` |
| `internal/adapter/hatchet/workflows/billing_sweep.go` | `billing-sweep` + `org-billing` fan-out (billing **and** reminders) | **Create** |
| `internal/adapter/hatchet/workflows/billing_cycle_runner.go` | `billing-cycle-runner` (one bounded cycle) | **Create** |
| `internal/adapter/hatchet/workflows/send_renewal_reminder.go` | `send-renewal-reminder` task (one reminder send) | **Create** |
| `internal/adapter/hatchet/hatchet.go` | worker wiring | **Modify**: construct/register new WFs + cron; drop `subscription-runner`; no-op `StartSubscriptionWorkflow`; pass `orgRepo` + `reminderResolver` |
| `internal/config/app.go` | DI wiring | **Modify**: construct `ReminderConfigService` + handler; pass `orgRepo` + resolver into `NewHatchetEngine` |
| `internal/adapter/hatchet/workflows/subscription_runner.go` | old immortal runner | **Delete** (after callers gone) |
| `internal/adapter/hatchet/workflows/subscription_charge_reminder.go` | reminder spawned by old runner | **Delete** (replaced by the new cron-driven `send-renewal-reminder`) |
| `internal/adapter/temporal/activities/order_activities.go` | Temporal activities | **Modify** (Task 9): add `ResolveReminderConfig` activity + resolver field |
| `internal/adapter/temporal/workflows/subscription_workflow.go` | Temporal durable runner | **Modify** (Task 9): per-offset reminders, config resolved per cycle |
| `internal/adapter/temporal/workflows/keys.go` | Temporal workflow ids | **Modify** (Task 9): `ReminderWorkflowID` per-(cycle,offset) |
| `docs/internal/README.md` | index | **Modify**: link this plan |

---

## Task 0: Audit charge-result handlers for idempotency

**Files:**
- Read: `internal/core/service/subscription.go` (the `HandleSubscriptionChargeSuccess` / `HandleSubscriptionChargeFailure` / `ChargeForBillingPeriod` impls)
- Read: `internal/core/service/dunning_orchestration.go` (campaign open on `subscription.payment.charge.failed`)

- [ ] **Step 1: Trace the two handlers.** Confirm whether re-invoking them for the *same* cycle (same `CyclesProcessed`) is safe: does success double-increment `CyclesProcessed`/`TotalRevenue`? Does failure open a second `DunningCampaign`?

- [ ] **Step 2: Record findings inline in this plan** (edit the checkbox note below) and, if NOT idempotent, add a guarded early-return keyed on cycle. Minimal guard sketch for success:

```go
// in HandleSubscriptionChargeSuccess, after loading the fresh subscription:
if input.Subscription.CyclesProcessed < current.CyclesProcessed {
    // this cycle was already applied (replay/retry) — no-op
    return current, nil
}
```

- [ ] **Step 3: Commit** any guards.

```bash
git add internal/core/service/
git commit -m "fix(subscription): make charge-result handlers idempotent per cycle"
```

> **Finding (fill in during Step 1):** _______________________________________________

---

## Task 1: `OrgRepository.ListIds` — enumerate tenants for fan-out

**Files:**
- Modify: `internal/core/port/repository.go:126-128` (the `OrgRepository` interface)
- Modify: `internal/adapter/postgres/org_repo.go`
- Test: `internal/adapter/postgres/org_repo_test.go` (create)

- [ ] **Step 1: Write the failing integration test**

Create `internal/adapter/postgres/org_repo_test.go`:

```go
//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"getpaidhq/internal/core/domain"

	"github.com/stretchr/testify/require"
)

func TestOrgRepo_ListIds(t *testing.T) {
	db := testDB(t)
	repo := NewOrgRepo(db)
	ctx := context.Background()

	orgId := uniqueOrg(t)
	t.Cleanup(func() { cleanupOrg(t, db, orgId) })

	_, err := repo.Create(ctx, domain.Org{
		Id:        orgId,
		Name:      "List Test",
		Status:    domain.OrgStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	ids, err := repo.ListIds(ctx)
	require.NoError(t, err)
	require.Contains(t, ids, orgId)
}
```

- [ ] **Step 2: Run it to verify it fails to compile** (method undefined)

Run: `go test -tags=integration ./internal/adapter/postgres/ -run TestOrgRepo_ListIds -v`
Expected: build error `repo.ListIds undefined`.

- [ ] **Step 3: Add to the interface**

In `internal/core/port/repository.go`, change the `OrgRepository` interface to:

```go
type OrgRepository interface {
	Create(ctx context.Context, entity domain.Org) (domain.Org, error)
	// ListIds returns every org id. Used by the billing sweep to fan out
	// one per-org billing run per tenant (tenant = the sharding axis).
	ListIds(ctx context.Context) ([]string, error)
}
```

- [ ] **Step 4: Implement it**

In `internal/adapter/postgres/org_repo.go`, add:

```go
func (r *OrgRepo) ListIds(ctx context.Context) ([]string, error) {
	var ids []string
	err := dbFromCtx(ctx, r.db).
		Model(&domain.Org{}).
		Where("status = ?", domain.OrgStatusActive).
		Pluck("id", &ids).Error
	return ids, err
}
```

> If `domain.Org` has no `Status`/`OrgStatusActive`, drop the `Where` and pluck all ids. Verify against `internal/core/domain/org.go` first.

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test -tags=integration ./internal/adapter/postgres/ -run TestOrgRepo_ListIds -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/core/port/repository.go internal/adapter/postgres/org_repo.go internal/adapter/postgres/org_repo_test.go
git commit -m "feat(repo): OrgRepository.ListIds for billing fan-out"
```

---

## Task 2: `SubscriptionRepository.FindDueForBilling` — the per-org due query

**Files:**
- Modify: `internal/core/port/repository.go:19-31` (the `SubscriptionRepository` interface)
- Modify: `internal/adapter/postgres/subscription_repo.go`
- Test: `internal/adapter/postgres/subscription_repo_test.go` (create if absent)

The "due" rule mirrors `domain.Subscription.GetNextChargeDate()`:
- `active`   → due when `renews_at  <= now`
- `past_due` → due when `next_retry <= now`
- `trial`    → due when `trial_ends_at <= now`

**Dependency:** the subscription date columns use `serializer:nulltime` (the in-flight `subscription.go` change + `internal/adapter/postgres/nulltime.go`), so unset dates are **NULL**. `col <= now` is already false for NULL, so unset rows are auto-excluded — **no epoch guard needed**.

- [ ] **Step 1: Write the failing integration test**

Create `internal/adapter/postgres/subscription_repo_test.go` (if it exists, append the test):

```go
//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"getpaidhq/internal/core/domain"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionRepo_FindDueForBilling(t *testing.T) {
	db := testDB(t)
	repo := NewSubscriptionRepo(db)
	ctx := context.Background()

	orgId := uniqueOrg(t)
	t.Cleanup(func() { cleanupOrg(t, db, orgId) })

	now := time.Now().UTC()
	mk := func(id string, status domain.SubscriptionStatus, renews time.Time) {
		_, err := repo.Create(ctx, domain.Subscription{
			OrgId: orgId, Id: id, Status: status,
			BillingInterval: domain.BillingIntervalMonth, BillingIntervalQty: 1,
			RenewsAt: renews, Currency: "USD", Amount: 1000,
			CreatedAt: now, UpdatedAt: now,
		})
		require.NoError(t, err)
	}

	mk("due-active", domain.SubscriptionStatusActive, now.Add(-time.Hour))   // due
	mk("future", domain.SubscriptionStatusActive, now.Add(48*time.Hour))     // not due
	mk("paused", domain.SubscriptionStatusPaused, now.Add(-time.Hour))       // excluded (status)

	due, err := repo.FindDueForBilling(ctx, orgId, now)
	require.NoError(t, err)

	ids := map[string]bool{}
	for _, s := range due {
		ids[s.Id] = true
	}
	require.True(t, ids["due-active"], "active+past renews_at should be due")
	require.False(t, ids["future"], "future renews_at should not be due")
	require.False(t, ids["paused"], "paused should be excluded")
}
```

> Verify `domain.BillingIntervalMonth` is the correct constant name in `internal/core/domain/subscription.go` (the switch uses string `"month"`); adjust the constant reference if needed.

- [ ] **Step 2: Run it to verify it fails** (method undefined)

Run: `go test -tags=integration ./internal/adapter/postgres/ -run TestSubscriptionRepo_FindDueForBilling -v`
Expected: build error `repo.FindDueForBilling undefined`.

- [ ] **Step 3: Add to the interface**

In `internal/core/port/repository.go`, add to `SubscriptionRepository`:

```go
	// FindDueForBilling returns running subscriptions in org whose next charge
	// date is at or before `now`, per Subscription.GetNextChargeDate() semantics.
	// Used by the per-org billing run to fan out one billing-cycle per due sub.
	FindDueForBilling(ctx context.Context, orgId string, now time.Time) ([]domain.Subscription, error)
```

Ensure `time` is imported in that file (it is, for other signatures — verify).

- [ ] **Step 4: Implement it**

In `internal/adapter/postgres/subscription_repo.go`, add:

```go
func (r *SubscriptionRepo) FindDueForBilling(ctx context.Context, orgId string, now time.Time) ([]domain.Subscription, error) {
	var subs []domain.Subscription
	// Unset date columns are NULL (serializer:nulltime maps zero time → NULL),
	// and `col <= now` is already false for NULL, so unset rows are auto-excluded.
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where(
			r.db.Where("status = ? AND renews_at <= ?", domain.SubscriptionStatusActive, now).
				Or("status = ? AND next_retry <= ?", domain.SubscriptionStatusPastDue, now).
				Or("status = ? AND trial_ends_at <= ?", domain.SubscriptionStatusTrial, now),
		).
		Find(&subs).Error
	return subs, err
}
```

> Confirm `OrgScope` exists in the postgres package (it's used by `List` in `customer_repo.go`). Confirm the gorm column names match the struct tags (`renews_at`, `next_retry`, `trial_ends_at`). The `serializer:nulltime` tags on these columns are a prerequisite (in-flight `subscription.go` change) — without NULL semantics this query would also match epoch-dated unset rows.

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test -tags=integration ./internal/adapter/postgres/ -run TestSubscriptionRepo_FindDueForBilling -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/core/port/repository.go internal/adapter/postgres/subscription_repo.go internal/adapter/postgres/subscription_repo_test.go
git commit -m "feat(repo): SubscriptionRepository.FindDueForBilling due-query"
```

---

## Task 2B: `SubscriptionRepository.FindUpcomingRenewals` — reminder-window query

Reminders need the set of subscriptions renewing *soon* (within the largest configured offset), so the sweep can decide per-stage which to remind. Active subs only; `renews_at` strictly in the future window `(now, now+within]`.

**Files:**
- Modify: `internal/core/port/repository.go` (`SubscriptionRepository`)
- Modify: `internal/adapter/postgres/subscription_repo.go`
- Test: `internal/adapter/postgres/subscription_repo_test.go` (append)

- [ ] **Step 1: Write the failing test** (append to `subscription_repo_test.go`)

```go
func TestSubscriptionRepo_FindUpcomingRenewals(t *testing.T) {
	db := testDB(t)
	repo := NewSubscriptionRepo(db)
	ctx := context.Background()

	orgId := uniqueOrg(t)
	t.Cleanup(func() { cleanupOrg(t, db, orgId) })

	now := time.Now().UTC()
	mk := func(id string, renews time.Time) {
		_, err := repo.Create(ctx, domain.Subscription{
			OrgId: orgId, Id: id, Status: domain.SubscriptionStatusActive,
			BillingInterval: domain.BillingIntervalMonth, BillingIntervalQty: 1,
			RenewsAt: renews, Currency: "USD", Amount: 1000, CreatedAt: now, UpdatedAt: now,
		})
		require.NoError(t, err)
	}
	mk("in-3-days", now.Add(72*time.Hour))   // inside 7d window
	mk("in-10-days", now.Add(240*time.Hour)) // outside 7d window
	mk("already-due", now.Add(-time.Hour))   // not upcoming (past)

	up, err := repo.FindUpcomingRenewals(ctx, orgId, now, 7*24*time.Hour)
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, s := range up {
		ids[s.Id] = true
	}
	require.True(t, ids["in-3-days"])
	require.False(t, ids["in-10-days"])
	require.False(t, ids["already-due"])
}
```

- [ ] **Step 2: Run to verify it fails** (undefined)

Run: `go test -tags=integration ./internal/adapter/postgres/ -run TestSubscriptionRepo_FindUpcomingRenewals -v`
Expected: build error `repo.FindUpcomingRenewals undefined`.

- [ ] **Step 3: Add to the interface** (in `SubscriptionRepository`)

```go
	// FindUpcomingRenewals returns active subscriptions whose renews_at falls in
	// (now, now+within]. The reminder sweep then picks per-offset stages from this set.
	FindUpcomingRenewals(ctx context.Context, orgId string, now time.Time, within time.Duration) ([]domain.Subscription, error)
```

- [ ] **Step 4: Implement it** (in `subscription_repo.go`)

```go
func (r *SubscriptionRepo) FindUpcomingRenewals(ctx context.Context, orgId string, now time.Time, within time.Duration) ([]domain.Subscription, error) {
	var subs []domain.Subscription
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("status = ? AND renews_at > ? AND renews_at <= ?",
			domain.SubscriptionStatusActive, now, now.Add(within)).
		Find(&subs).Error
	return subs, err
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test -tags=integration ./internal/adapter/postgres/ -run TestSubscriptionRepo_FindUpcomingRenewals -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/core/port/repository.go internal/adapter/postgres/subscription_repo.go internal/adapter/postgres/subscription_repo_test.go
git commit -m "feat(repo): SubscriptionRepository.FindUpcomingRenewals reminder-window query"
```

---

## Task 3: Workflow input + run-key for per-org fan-out

**Files:**
- Modify: `internal/adapter/hatchet/workflows/types.go`
- Modify: `internal/adapter/hatchet/workflows/keys.go`

- [ ] **Step 1: Add the input struct**

In `internal/adapter/hatchet/workflows/types.go`, add:

```go
// OrgBillingInput is the input for the per-org billing fan-out run.
type OrgBillingInput struct {
	OrgId string `json:"org_id"`
}
```

- [ ] **Step 2: Add the run key**

In `internal/adapter/hatchet/workflows/keys.go`, add:

```go
// OrgBillingRunKey dedups the per-org billing fan-out within a single sweep
// bucket (the truncated-hour timestamp), so an accidental double-sweep in the
// same hour doesn't double-spawn an org's billing run.
func OrgBillingRunKey(orgId string, bucket time.Time) string {
	return fmt.Sprintf("orgbilling_%s_%s", orgId, bucket.UTC().Format("2006010215"))
}

// ReminderStageRunKey dedups a renewal reminder to exactly once per
// (subscription, cycle, offset-stage). The sweep may re-spawn this every hour
// the sub is inside the stage window; identical keys collapse (USE_EXISTING),
// so the reminder sends once per stage per cycle and self-heals across missed
// ticks. `cycle` is CyclesProcessed; the offset label distinguishes stages
// (e.g. "168h" vs "24h").
func ReminderStageRunKey(orgId, subscriptionId string, cycle int, offset time.Duration) string {
	return fmt.Sprintf("reminder_%s_%s_%s_%s", orgId, subscriptionId, strconv.Itoa(cycle), offset.String())
}
```

(`fmt` and `time` are already imported in `keys.go`; **add `strconv`** to its import block.)

- [ ] **Step 3: Build to verify it compiles**

Run: `go build ./internal/adapter/hatchet/...`
Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/hatchet/workflows/types.go internal/adapter/hatchet/workflows/keys.go
git commit -m "feat(hatchet): OrgBillingInput + OrgBillingRunKey"
```

---

## Task 4: `billing-cycle-runner` — one bounded durable cycle

This lifts the per-cycle body out of the old immortal runner (`subscription_runner.go:122-160`): spawn the charge DAG, optionally wait ≤1h for the webhook, handle the result, then **exit**. Bounded → ages out of retention. The ≤1h webhook wait needs an eviction policy (TTL < execution timeout) so it isn't reaped by the 5-min default.

**Files:**
- Create: `internal/adapter/hatchet/workflows/billing_cycle_runner.go`

- [ ] **Step 1: Write the workflow**

Create `internal/adapter/hatchet/workflows/billing_cycle_runner.go`:

```go
package workflows

import (
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewBillingCycleRunnerWorkflow builds a BOUNDED, one-shot durable task that
// processes exactly one billing cycle and exits — unlike the retired immortal
// subscription-runner. Because it always completes, its durable-event-log rows
// live briefly in their birth-date partition and are dropped cleanly by
// retention. See docs/internal/durable-runner-timeouts.md.
//
// Flow (mirrors the old runner's per-cycle body):
//  1. Run the billing-cycle charge DAG (idempotent via billing_<org>_<sub>_<cycle>).
//  2. If the charge is Pending, wait up to 1h for the PSP webhook event.
//  3. Hand the final ChargeResult to the subscription service, which advances
//     state (RenewsAt/CyclesProcessed) or opens dunning.
//
// The ≤1h wait needs eviction (TTL < execution timeout) so it isn't reaped by
// the 5-minute default execution timeout. All side effects must be idempotent
// (see plan Task 0): eviction/restart replays this function from the top.
func NewBillingCycleRunnerWorkflow(client *hatchet.Client, subscriptionService port.SubscriptionService) *hatchet.StandaloneTask {
	return client.NewStandaloneDurableTask("billing-cycle-runner",
		func(ctx hatchet.DurableContext, sub domain.Subscription) (domain.Subscription, error) {
			// 1. Charge (child DAG; durable + idempotent by run key → replay-safe).
			billingRes, err := client.Run(ctx, "billing-cycle", BillingCycleInput{Subscription: sub},
				hatchet.WithRunKey(BillingRunKey(sub.OrgId, sub.Id, sub.CyclesProcessed)),
				hatchet.WithRunMetadata(map[string]string{
					"orgId": sub.OrgId, "subscriptionId": sub.Id,
				}),
			)
			if err != nil {
				// Infra failure (e.g. no gateway). Non-fatal: log + exit; the next
				// hourly sweep re-selects this sub (still due) and retries.
				return sub, err
			}

			var chargeResult domain.ChargeResult
			if err := billingRes.TaskOutput("charge-customer").Into(&chargeResult); err != nil {
				return sub, err
			}

			// 2. Pending → wait up to 1h for the webhook to deliver the final status.
			if chargeResult.Status == domain.PaymentStatusPending {
				webhookKey := WebhookEventKey(sub.OrgId, sub.Id)
				wr, err := ctx.WaitFor(hatchet.OrCondition(
					hatchet.SleepCondition(1*time.Hour),
					hatchet.UserEventCondition(webhookKey, ""),
				))
				if err == nil && containsKey(waitedKeys(wr), webhookKey) {
					var fromWebhook domain.ChargeResult
					if err := unmarshalWaited(wr, webhookKey, &fromWebhook); err == nil {
						chargeResult = fromWebhook
					}
				}
			}

			// 3. Apply the result (idempotent per cycle — see plan Task 0).
			input := domain.SubscriptionChargeInput{Subscription: sub, ChargeResult: chargeResult}
			if chargeResult.Status == domain.PaymentStatusSucceeded {
				return subscriptionService.HandleSubscriptionChargeSuccess(ctx, input)
			}
			return subscriptionService.HandleSubscriptionChargeFailure(ctx, input)
		},
		hatchet.WithExecutionTimeout(5*time.Minute), // > eviction TTL
		hatchet.WithEvictionPolicy(&hatchet.EvictionPolicy{
			TTL:                   30 * time.Second, // evict during the ≤1h webhook wait
			AllowCapacityEviction: true,
		}),
	)
}
```

> `containsKey`, `waitedKeys`, `unmarshalWaited` already exist in the workflows package (used by `subscription_runner.go` — they survive its deletion only if defined elsewhere; **verify** their definitions are NOT inside `subscription_runner.go`. If they are, move them into `keys.go` or a new `wait_helpers.go` in Task 7 before deleting the runner.)

- [ ] **Step 2: Build to verify it compiles**

Run: `go build ./internal/adapter/hatchet/...`
Expected: success (or a clear error pointing at a helper that must be relocated — handle in Task 7).

- [ ] **Step 3: Commit**

```bash
git add internal/adapter/hatchet/workflows/billing_cycle_runner.go
git commit -m "feat(hatchet): billing-cycle-runner (bounded one-cycle durable task)"
```

---

## Task 4B: `ReminderConfig` domain type + `SettingRepository.Upsert`

Reminder config is **per-tenant**, stored in the existing `settings` table (`Setting{OrgId, ParentId, Id, Value(JSON)}`), resolved with a default fallback — mirroring the `DunningConfig` precedent (`DunningService.ResolveConfig` → `DefaultDunningConfig()`). No env var.

**Files:**
- Create: `internal/core/domain/reminder_config.go`
- Modify: `internal/core/port/repository.go` (`SettingRepository`)
- Modify: `internal/adapter/postgres/setting_repo.go`
- Test: `internal/core/domain/reminder_config_test.go` (create)

- [ ] **Step 1: Write the failing domain test**

Create `internal/core/domain/reminder_config_test.go`:

```go
package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReminderConfig_RoundTrip(t *testing.T) {
	cfg := ReminderConfig{Enabled: true, Offsets: []time.Duration{168 * time.Hour, 24 * time.Hour}}
	raw, err := cfg.Marshal()
	require.NoError(t, err)

	got, err := ParseReminderConfig(raw)
	require.NoError(t, err)
	require.True(t, got.Enabled)
	require.Equal(t, []time.Duration{168 * time.Hour, 24 * time.Hour}, got.Offsets)
}

func TestParseReminderConfig_EmptyIsDefault(t *testing.T) {
	got, err := ParseReminderConfig("")
	require.NoError(t, err)
	require.Equal(t, DefaultReminderConfig(), got)
}
```

- [ ] **Step 2: Run to verify it fails** (undefined symbols)

Run: `go test ./internal/core/domain/ -run TestReminderConfig -v`
Expected: build error (`ReminderConfig` / `ParseReminderConfig` / `DefaultReminderConfig` undefined).

- [ ] **Step 3: Implement the domain type**

Create `internal/core/domain/reminder_config.go`:

```go
package domain

import (
	"encoding/json"
	"time"
)

// Setting coordinates for the per-org renewal-reminder config.
const (
	ReminderConfigSettingParent = "billing"
	ReminderConfigSettingId     = "renewal_reminders"
)

// ReminderConfig is the resolved per-tenant renewal-reminder policy.
type ReminderConfig struct {
	Enabled bool            `json:"enabled"`
	Offsets []time.Duration `json:"-"` // lead times before renewal, e.g. 168h, 24h
}

// reminderConfigJSON is the persisted shape (durations as human strings like
// "168h", not int64 nanoseconds) for a readable/editable setting value.
type reminderConfigJSON struct {
	Enabled bool     `json:"enabled"`
	Offsets []string `json:"offsets"`
}

// DefaultReminderConfig is the fallback when an org has no reminder setting:
// one reminder 7 days before renewal. Tenants override (incl. disable) via the
// reminder-config endpoint. Mirrors DefaultDunningConfig()'s role.
func DefaultReminderConfig() ReminderConfig {
	return ReminderConfig{Enabled: true, Offsets: []time.Duration{7 * 24 * time.Hour}}
}

// Marshal renders the config to the persisted JSON string (durations as strings).
func (c ReminderConfig) Marshal() (string, error) {
	dto := reminderConfigJSON{Enabled: c.Enabled}
	for _, d := range c.Offsets {
		dto.Offsets = append(dto.Offsets, d.String())
	}
	b, err := json.Marshal(dto)
	return string(b), err
}

// ParseReminderConfig parses a persisted value; empty input returns the default.
func ParseReminderConfig(raw string) (ReminderConfig, error) {
	if raw == "" {
		return DefaultReminderConfig(), nil
	}
	var dto reminderConfigJSON
	if err := json.Unmarshal([]byte(raw), &dto); err != nil {
		return ReminderConfig{}, err
	}
	cfg := ReminderConfig{Enabled: dto.Enabled}
	for _, s := range dto.Offsets {
		d, err := time.ParseDuration(s)
		if err != nil {
			return ReminderConfig{}, err
		}
		cfg.Offsets = append(cfg.Offsets, d)
	}
	return cfg, nil
}
```

- [ ] **Step 4: Run the domain test to verify it passes**

Run: `go test ./internal/core/domain/ -run TestReminderConfig -v`
Expected: PASS.

- [ ] **Step 5: Add `Upsert` to `SettingRepository`** (re-setting must not PK-conflict; `Create` is insert-only)

In `internal/core/port/repository.go`, add to `SettingRepository`:

```go
	// Upsert creates or replaces a setting by its (OrgId, ParentId, Id) key.
	Upsert(ctx context.Context, entity domain.Setting) (domain.Setting, error)
```

In `internal/adapter/postgres/setting_repo.go`, implement it with GORM's on-conflict upsert:

```go
func (r *SettingRepo) Upsert(ctx context.Context, entity domain.Setting) (domain.Setting, error) {
	err := dbFromCtx(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "org_id"}, {Name: "parent_id"}, {Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "updated_at"}),
		}).
		Create(&entity).Error
	if err != nil {
		return domain.Setting{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.ParentId, entity.Id)
}
```

Add `"gorm.io/gorm/clause"` to the imports in `setting_repo.go`.

- [ ] **Step 6: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 7: Commit**

```bash
git add internal/core/domain/reminder_config.go internal/core/domain/reminder_config_test.go internal/core/port/repository.go internal/adapter/postgres/setting_repo.go
git commit -m "feat(billing): ReminderConfig domain type + SettingRepository.Upsert"
```

---

## Task 4C: `ReminderConfigService` + `port.ReminderConfigResolver`

A small service resolves/sets the per-tenant config via `SettingRepository`, with the default fallback. The narrow `ReminderConfigResolver` port is what `org-billing` depends on (keeps the workflow's dependency minimal).

**Files:**
- Modify: `internal/core/port/service.go` (add `ReminderConfigResolver`)
- Create: `internal/core/service/reminder_config.go`
- Test: `internal/core/service/reminder_config_test.go` (create)

- [ ] **Step 1: Add the narrow resolver port**

In `internal/core/port/service.go`, add:

```go
// ReminderConfigResolver resolves the per-tenant renewal-reminder policy.
// The billing sweep depends only on this read method.
type ReminderConfigResolver interface {
	ResolveReminderConfig(ctx context.Context, orgId string) (domain.ReminderConfig, error)
}
```

- [ ] **Step 2: Write the failing service test**

Create `internal/core/service/reminder_config_test.go`. Use the package's existing mock/fake style for `port.SettingRepository` (model on a sibling `*_test.go` that already fakes a repo). The test asserts: missing setting → `DefaultReminderConfig()`; present setting → parsed offsets.

```go
package service

import (
	"context"
	"testing"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/lib"

	"github.com/stretchr/testify/require"
)

func TestReminderConfigService_Resolve_DefaultWhenMissing(t *testing.T) {
	repo := newFakeSettingRepo() // returns NotFound for unknown keys — mirror an existing fake
	svc := NewReminderConfigService(repo, lib.NewNoopLogger())

	cfg, err := svc.ResolveReminderConfig(context.Background(), "org_x")
	require.NoError(t, err)
	require.Equal(t, domain.DefaultReminderConfig(), cfg)
}

func TestReminderConfigService_SetThenResolve(t *testing.T) {
	repo := newFakeSettingRepo()
	svc := NewReminderConfigService(repo, lib.NewNoopLogger())

	want := domain.ReminderConfig{Enabled: true, Offsets: []time.Duration{24 * time.Hour}}
	require.NoError(t, svc.SetReminderConfig(context.Background(), "org_x", want))

	got, err := svc.ResolveReminderConfig(context.Background(), "org_x")
	require.NoError(t, err)
	require.Equal(t, want, got)
}
```

> `newFakeSettingRepo` / `lib.NewNoopLogger`: use whatever the service package already uses for fakes and a logger in its tests (grep `internal/core/service/*_test.go` for the established pattern; if there's a shared fake, reuse it — do not invent a new mock framework).

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/core/service/ -run TestReminderConfigService -v`
Expected: build error (`NewReminderConfigService` undefined).

- [ ] **Step 4: Implement the service**

Create `internal/core/service/reminder_config.go`:

```go
package service

import (
	"context"
	"errors"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type ReminderConfigService struct {
	settings port.SettingRepository
	logger   port.Logger
}

func NewReminderConfigService(settings port.SettingRepository, logger port.Logger) *ReminderConfigService {
	return &ReminderConfigService{settings: settings, logger: logger}
}

// ResolveReminderConfig returns the org's reminder policy, or the default when
// no setting exists (mirrors DunningService.ResolveConfig).
func (s *ReminderConfigService) ResolveReminderConfig(ctx context.Context, orgId string) (domain.ReminderConfig, error) {
	setting, err := s.settings.FindById(ctx, orgId, domain.ReminderConfigSettingParent, domain.ReminderConfigSettingId)
	if err != nil {
		// Not-found → default. Translate to the package's not-found sentinel.
		if errors.Is(err, lib.ErrNotFound) {
			return domain.DefaultReminderConfig(), nil
		}
		s.logger.Error("ResolveReminderConfig failed, using default", "orgId", orgId, "err", err.Error())
		return domain.DefaultReminderConfig(), nil
	}
	cfg, err := domain.ParseReminderConfig(setting.Value)
	if err != nil {
		s.logger.Error("invalid reminder config, using default", "orgId", orgId, "err", err.Error())
		return domain.DefaultReminderConfig(), nil
	}
	return cfg, nil
}

// SetReminderConfig upserts the org's reminder policy.
func (s *ReminderConfigService) SetReminderConfig(ctx context.Context, orgId string, cfg domain.ReminderConfig) error {
	value, err := cfg.Marshal()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = s.settings.Upsert(ctx, domain.Setting{
		OrgId:     orgId,
		ParentId:  domain.ReminderConfigSettingParent,
		Id:        domain.ReminderConfigSettingId,
		Type:      "json",
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	})
	return err
}
```

> `lib.ErrNotFound`: confirm the package's not-found sentinel name (grep `lib.Err` / `translateErr`); the postgres `FindById` returns it via `translateErr`. Adjust the `errors.Is` target to match.

- [ ] **Step 5: Run the service test to verify it passes**

Run: `go test ./internal/core/service/ -run TestReminderConfigService -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/core/port/service.go internal/core/service/reminder_config.go internal/core/service/reminder_config_test.go
git commit -m "feat(billing): ReminderConfigService (per-tenant resolve/set)"
```

---

## Task 4D: `send-renewal-reminder` workflow

**Files:**
- Create: `internal/adapter/hatchet/workflows/send_renewal_reminder.go`

- [ ] **Step 1: Write the workflow**

Create `internal/adapter/hatchet/workflows/send_renewal_reminder.go`:

```go
package workflows

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// RenewalReminderInput carries the subscription whose renewal reminder should
// be sent. (Named distinctly from the old runner's `ReminderInput`, which still
// exists in subscription_charge_reminder.go until Task 7 — avoids a duplicate
// declaration so every task builds green.)
type RenewalReminderInput struct {
	Subscription domain.Subscription `json:"subscription"`
}

// NewSendRenewalReminderWorkflow builds a short, non-durable task that sends one
// renewal reminder. The per-(cycle, offset) run key (ReminderStageRunKey) makes
// each stage fire once per cycle, so this task itself just performs the send.
func NewSendRenewalReminderWorkflow(client *hatchet.Client, subscriptionService port.SubscriptionService) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("send-renewal-reminder",
		func(ctx hatchet.Context, in RenewalReminderInput) (struct{}, error) {
			err := subscriptionService.SendRenewalReminder(ctx, in.Subscription.OrgId, in.Subscription.Id)
			return struct{}{}, err
		},
	)
}
```

> `RenewalReminderInput` has a distinct name from the old runner's `ReminderInput` (in `subscription_charge_reminder.go`, deleted in Task 7), so there is no duplicate-declaration collision in the window before Task 7 — the build stays green at every step.

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: success. `RenewalReminderInput` is distinctly named, so there is no collision with the old `ReminderInput` (which still exists until Task 7).

- [ ] **Step 3: Commit**

```bash
git add internal/adapter/hatchet/workflows/send_renewal_reminder.go
git commit -m "feat(hatchet): send-renewal-reminder task"
```

---

## Task 4E: Tenant-facing reminder-config endpoint (GET/PUT)

So merchants set their own reminder policy. Model exactly on an existing fuego handler (`internal/adapter/http/dunning_handler.go` / `psp_handler.go`): read the org from the auth context (`handler.AuthUserFrom(c)`), call the service, return the `ApiError` envelope.

**Files:**
- Create: `internal/adapter/http/reminder_config_handler.go`
- Modify: `internal/config/server.go` (register the route group)
- Modify: `internal/config/app.go` (construct the handler + service)

- [ ] **Step 1: Write the handler**

Create `internal/adapter/http/reminder_config_handler.go` (adjust types/option calls to match the sibling handlers exactly):

```go
package http

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/service"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
)

type ReminderConfigHandler struct {
	service *service.ReminderConfigService
	logger  port.Logger
}

func NewReminderConfigHandler(s *service.ReminderConfigService, logger port.Logger) *ReminderConfigHandler {
	return &ReminderConfigHandler{service: s, logger: logger}
}

// ReminderConfigDTO is the wire shape (durations as strings, e.g. "168h").
type ReminderConfigDTO struct {
	Enabled bool     `json:"enabled"`
	Offsets []string `json:"offsets" validate:"dive,required"`
}

func (h *ReminderConfigHandler) Register(g *fuego.Server) {
	grp := fuego.Group(g, "/api/billing/reminder-config")
	fuego.Get(grp, "", h.Get, option.Summary("Get renewal reminder config"))
	fuego.Put(grp, "", h.Put, option.Summary("Set renewal reminder config"))
}

func (h *ReminderConfigHandler) Get(c fuego.ContextNoBody) (ReminderConfigDTO, error) {
	user := AuthUserFrom(c)
	cfg, err := h.service.ResolveReminderConfig(c.Context(), user.OrgId)
	if err != nil {
		return ReminderConfigDTO{}, err
	}
	return toReminderDTO(cfg), nil
}

func (h *ReminderConfigHandler) Put(c fuego.ContextWithBody[ReminderConfigDTO]) (ReminderConfigDTO, error) {
	body, err := c.Body()
	if err != nil {
		return ReminderConfigDTO{}, err
	}
	cfg, err := fromReminderDTO(body)
	if err != nil {
		return ReminderConfigDTO{}, fuego.BadRequestError{Title: "invalid offset duration", Detail: err.Error()}
	}
	user := AuthUserFrom(c)
	if err := h.service.SetReminderConfig(c.Context(), user.OrgId, cfg); err != nil {
		return ReminderConfigDTO{}, err
	}
	return toReminderDTO(cfg), nil
}

func toReminderDTO(cfg domain.ReminderConfig) ReminderConfigDTO {
	dto := ReminderConfigDTO{Enabled: cfg.Enabled}
	for _, d := range cfg.Offsets {
		dto.Offsets = append(dto.Offsets, d.String())
	}
	return dto
}

func fromReminderDTO(dto ReminderConfigDTO) (domain.ReminderConfig, error) {
	cfg := domain.ReminderConfig{Enabled: dto.Enabled}
	for _, s := range dto.Offsets {
		d, err := time.ParseDuration(s)
		if err != nil {
			return domain.ReminderConfig{}, err
		}
		cfg.Offsets = append(cfg.Offsets, d)
	}
	return cfg, nil
}
```

> Verify against a sibling handler: exact import path/name for `port` (logger), `AuthUserFrom` location, the `fuego.Group`/`option`/error types, and how `user.OrgId` is named on `port.AuthUser`. Add `"time"` to imports. If handlers must call `authzEngine` before mutations (per CLAUDE.md), inject and call it in `Put` like `OrderHandler` does.

- [ ] **Step 2: Construct + register in app.go / server.go**

In `internal/config/app.go`, construct `reminderConfigService := service.NewReminderConfigService(settingRepo, logger)` (near the other services) and `handler.NewReminderConfigHandler(reminderConfigService, logger)`, then register it where the other handlers are registered in `internal/config/server.go` (follow the existing `Register`/route-group pattern there). Keep a reference to `reminderConfigService` — Task 6 also injects it into the Hatchet engine.

- [ ] **Step 3: Build + smoke the route**

Run: `go build ./... && go run ./cmd/openapi-export` (regenerates `openapi.yml`; confirms the typed handler is valid).
Expected: success; `openapi.yml` gains `/api/billing/reminder-config`.

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/http/reminder_config_handler.go internal/config/server.go internal/config/app.go openapi.yml
git commit -m "feat(http): tenant renewal-reminder config endpoint"
```

---

## Task 5: `billing-sweep` (cron) + `org-billing` fan-out (billing **and** reminders)

**Files:**
- Create: `internal/adapter/hatchet/workflows/billing_sweep.go`

- [ ] **Step 1: Write both workflows**

Create `internal/adapter/hatchet/workflows/billing_sweep.go`:

```go
package workflows

import (
	"time"

	"getpaidhq/internal/core/port"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewBillingSweepWorkflow builds the hourly cron entrypoint. It does NO
// subscription work itself: it lists org ids and spawns one org-billing run
// per tenant (the tenant is the sharding axis — a whale org can't block
// others). Modeled on Lago's SubscriptionsBillerJob. Non-durable: a fresh run
// each tick, plain time.Now() is fine (no replay).
func NewBillingSweepWorkflow(client *hatchet.Client, orgRepo port.OrgRepository, logger port.Logger) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("billing-sweep",
		func(ctx hatchet.Context, _ struct{}) (struct{}, error) {
			ids, err := orgRepo.ListIds(ctx)
			if err != nil {
				logger.Error("billing-sweep: ListIds failed", "err", err.Error())
				return struct{}{}, err
			}
			bucket := time.Now().UTC().Truncate(time.Hour)
			for _, orgId := range ids {
				if _, err := client.RunNoWait(ctx, "org-billing", OrgBillingInput{OrgId: orgId},
					hatchet.WithRunKey(OrgBillingRunKey(orgId, bucket)),
					hatchet.WithRunMetadata(map[string]string{"orgId": orgId}),
				); err != nil {
					logger.Error("billing-sweep: spawn org-billing failed", "orgId", orgId, "err", err.Error())
					// continue: one org's failure must not stop the rest
				}
			}
			logger.Infof("billing-sweep fanned out to %d orgs", len(ids))
			return struct{}{}, nil
		},
		hatchet.WithCron("10 * * * *"), // hourly at :10, mirrors Lago's bill_customers cadence
	)
}

// NewOrgBillingWorkflow builds the per-org fan-out. It does two scans over the
// org's subscriptions each tick: (1) due-for-billing → spawn billing-cycle-runner;
// (2) upcoming renewals → spawn send-renewal-reminder per configured offset stage.
// Both idempotent via run keys. Non-durable: plain time.Now() is fine.
// The reminder policy is resolved PER TENANT (reminderResolver, backed by the
// settings table); a disabled or offset-less config ⇒ no reminders.
func NewOrgBillingWorkflow(client *hatchet.Client, subRepo port.SubscriptionRepository, reminderResolver port.ReminderConfigResolver, logger port.Logger) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("org-billing",
		func(ctx hatchet.Context, in OrgBillingInput) (struct{}, error) {
			now := time.Now().UTC()

			// --- (1) Billing fan-out ---
			due, err := subRepo.FindDueForBilling(ctx, in.OrgId, now)
			if err != nil {
				logger.Error("org-billing: FindDueForBilling failed", "orgId", in.OrgId, "err", err.Error())
				return struct{}{}, err
			}
			for _, sub := range due {
				if _, err := client.RunNoWait(ctx, "billing-cycle-runner", sub,
					hatchet.WithRunKey(BillingRunKey(sub.OrgId, sub.Id, sub.CyclesProcessed)),
					hatchet.WithRunMetadata(map[string]string{"orgId": sub.OrgId, "subscriptionId": sub.Id}),
				); err != nil {
					logger.Error("org-billing: spawn billing-cycle-runner failed",
						"orgId", sub.OrgId, "subscriptionId", sub.Id, "err", err.Error())
				}
			}

			// --- (2) Reminder fan-out (per-tenant config; default fallback) ---
			cfg, err := reminderResolver.ResolveReminderConfig(ctx, in.OrgId)
			if err != nil {
				logger.Error("org-billing: ResolveReminderConfig failed", "orgId", in.OrgId, "err", err.Error())
			} else if cfg.Enabled && len(cfg.Offsets) > 0 {
				maxOffset := cfg.Offsets[0]
				for _, d := range cfg.Offsets {
					if d > maxOffset {
						maxOffset = d
					}
				}
				upcoming, err := subRepo.FindUpcomingRenewals(ctx, in.OrgId, now, maxOffset)
				if err != nil {
					logger.Error("org-billing: FindUpcomingRenewals failed", "orgId", in.OrgId, "err", err.Error())
				} else {
					for _, sub := range upcoming {
						for _, offset := range cfg.Offsets {
							// Stage is active once we've crossed (renews_at - offset).
							// Re-spawning every tick is fine: the run key dedups to one send.
							if now.Before(sub.RenewsAt.Add(-offset)) {
								continue
							}
							if _, err := client.RunNoWait(ctx, "send-renewal-reminder", RenewalReminderInput{Subscription: sub},
								hatchet.WithRunKey(ReminderStageRunKey(sub.OrgId, sub.Id, sub.CyclesProcessed, offset)),
								hatchet.WithRunMetadata(map[string]string{
									"orgId": sub.OrgId, "subscriptionId": sub.Id, "reminderOffset": offset.String(),
								}),
							); err != nil {
								logger.Error("org-billing: spawn send-renewal-reminder failed",
									"orgId", sub.OrgId, "subscriptionId", sub.Id, "err", err.Error())
							}
						}
					}
				}
			}

			if len(due) > 0 {
				logger.Infof("org-billing[%s] spawned %d billing-cycle-runner(s)", in.OrgId, len(due))
			}
			return struct{}{}, nil
		},
	)
}
```

> The `billing-cycle-runner` run key reuses `BillingRunKey(org,sub,cycle)` — same key as the inner charge DAG. That's intentional and safe: Hatchet namespaces run keys by workflow name, so identical keys across `billing-cycle-runner` and `billing-cycle` do not collide; both dedup per cycle.
>
> **Reminder dedup correctness:** `ReminderStageRunKey` includes `CyclesProcessed` and the offset, so each (cycle, stage) sends once. A longer-lead stage (e.g. 168h) first qualifies days before a shorter one (24h); its key was created at the first qualifying tick and dedups on every subsequent tick, so when 24h later becomes active only *its* key is new → exactly the staged behavior. Run records persist within the retention window (weeks), far longer than any reminder lead, so dedup holds across the whole window.

- [ ] **Step 2: Build to verify it compiles**

Run: `go build ./internal/adapter/hatchet/...`
Expected: success. If `hatchet.Context`/`NewStandaloneTask` signatures differ, check `sdks/go/client.go:609` and the existing DAG task signature in `billing_cycle.go` and adjust.

- [ ] **Step 3: Commit**

```bash
git add internal/adapter/hatchet/workflows/billing_sweep.go
git commit -m "feat(hatchet): billing-sweep cron + org-billing fan-out"
```

---

## Task 6: Wire the new workflows; register cron; no-op the old start; pass orgRepo

**Files:**
- Modify: `internal/adapter/hatchet/hatchet.go` (constructor signature ~line 40-55, registration block 70-90, `StartSubscriptionWorkflow` ~199)
- Modify: `internal/config/app.go:189` (`NewHatchetEngine` call)

- [ ] **Step 1: Add `orgRepo` + `reminderResolver` to the engine constructor**

In `internal/adapter/hatchet/hatchet.go`, add `orgRepo port.OrgRepository` and `reminderResolver port.ReminderConfigResolver` to `NewHatchetEngine`'s parameter list (place `orgRepo` next to `subscriptionRepo`). Confirm `port` is imported.

- [ ] **Step 2: Construct + register the new workflows; drop the old two**

In the registration block (`hatchet.go:70-90`), replace the `subscriptionRunnerWF` and `reminderWF` lines with the new workflows:

```go
	billingCycleWF := hatchetwf.NewBillingCycleWorkflow(c, subscriptionService)
	billingCycleRunnerWF := hatchetwf.NewBillingCycleRunnerWorkflow(c, subscriptionService)
	orgBillingWF := hatchetwf.NewOrgBillingWorkflow(c, subscriptionRepo, reminderResolver, logger)
	billingSweepWF := hatchetwf.NewBillingSweepWorkflow(c, orgRepo, logger)
	sendReminderWF := hatchetwf.NewSendRenewalReminderWorkflow(c, subscriptionService)
	dunningAttemptWF := hatchetwf.NewDunningAttemptWorkflow(c, dunningSteps)
	dunningRunnerWF := hatchetwf.NewDunningRunnerWorkflow(c, dunningSteps)
```

And update `hatchet.WithWorkflows(...)` to register `billingCycleWF, billingCycleRunnerWF, orgBillingWF, billingSweepWF, sendReminderWF` (plus the unchanged payment/webhook/dunning ones) and **remove** `subscriptionRunnerWF` and `reminderWF` (the old per-charge reminder).

- [ ] **Step 3: No-op `StartSubscriptionWorkflow`**

Replace the body of `StartSubscriptionWorkflow` (`hatchet.go:~198-201`) with:

```go
func (h Hatchet) StartSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	// No-op under the cron + fan-out billing model: a newly active/trialing
	// subscription is picked up by the next hourly billing-sweep when its
	// RenewsAt/NextRetryAt/TrialEndsAt falls due. The immortal per-subscription
	// runner has been retired (see docs/internal/subscriptions-on-hatchet.md).
	h.logger.Debugf("StartSubscriptionWorkflow no-op (cron drives billing) org=%s sub=%s", sub.OrgId, sub.Id)
	return nil
}
```

Leave `UpdateSubscriptionWorkflow` / `CancelSubscriptionWorkflow` / `SignalSubscriptionWorkflow` as no-ops too if they only fed the runner (verify each pushes events only the runner consumed; if so, make them log-and-return). The `SubscriptionEventBridge` may still publish — harmless.

- [ ] **Step 4: Pass `orgRepo` + reminder resolver in app.go**

In `internal/config/app.go:189`, add `orgRepo` and `reminderConfigService` (the `*ReminderConfigService` built in Task 4E, which satisfies `port.ReminderConfigResolver`) to the `hatchet.NewHatchetEngine(...)` argument list at the matching positions. `orgRepo` already exists (`app.go:101`). Ensure `reminderConfigService` is constructed **before** the engine wiring block.

- [ ] **Step 5: Build the whole app**

Run: `go build ./...`
Expected: success. Fix any signature mismatches.

- [ ] **Step 6: Commit**

```bash
git add internal/adapter/hatchet/hatchet.go internal/config/app.go
git commit -m "feat(hatchet): register cron billing fan-out, retire subscription-runner start"
```

---

## Task 7: Delete the retired runner + reminder; relocate shared helpers

**Files:**
- Delete: `internal/adapter/hatchet/workflows/subscription_runner.go`
- Delete: `internal/adapter/hatchet/workflows/subscription_charge_reminder.go`
- Possibly Create: `internal/adapter/hatchet/workflows/wait_helpers.go` (if helpers lived in the runner)

- [ ] **Step 1: Relocate shared helpers if needed**

If `go build ./...` in Task 6 still passed, the helpers (`containsKey`, `waitedKeys`, `unmarshalWaited`, `isTerminalStatus`) are referenced by the new runner. Grep where they are defined:

Run: `grep -rn "func waitedKeys\|func unmarshalWaited\|func containsKey\|func isTerminalStatus" internal/adapter/hatchet/workflows/`

If any are defined in `subscription_runner.go`, cut them into a new `internal/adapter/hatchet/workflows/wait_helpers.go` (package `workflows`) before deleting.

> **`ReminderInput` note:** the new reminder workflow uses `RenewalReminderInput` (Task 4D) — a distinct type — so deleting `subscription_charge_reminder.go` (which owns the old Hatchet `ReminderInput`) causes no collision and leaves no dangling references. After deletion, `grep -rn "\bReminderInput\b\|SubscriptionChargeReminder\|NewSubscriptionChargeReminderWorkflow" internal/adapter/hatchet/` should return nothing (only `RenewalReminderInput` remains).

- [ ] **Step 2: Delete the two files**

```bash
git rm internal/adapter/hatchet/workflows/subscription_runner.go
git rm internal/adapter/hatchet/workflows/subscription_charge_reminder.go
```

- [ ] **Step 3: Build + vet to confirm nothing references them**

Run: `go build ./... && go vet ./internal/adapter/hatchet/...`
Expected: success. Resolve any `undefined: ReminderInput` / `NewSubscriptionRunnerWorkflow` references (there should be none after Task 6).

- [ ] **Step 4: Run the full unit suite**

Run: `go test ./...`
Expected: PASS (integration tests excluded by default; they ran in Tasks 1-2).

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(hatchet): delete immortal subscription-runner + reminder"
```

---

## Task 8: End-to-end smoke verification (manual, local stack)

**Files:** none (operational check).

- [ ] **Step 1: Ensure the local stack + a seeded PSP gateway exist** (see [org-seed-data.md](org-seed-data.md) — `billing-cycle` needs a `gateways` + `settings` row or every charge errors).

- [ ] **Step 2: Set a due subscription.** In the local DB, set one active subscription's `renews_at` to a past timestamp.

- [ ] **Step 3: Trigger the sweep without waiting an hour.** In the Hatchet UI (`http://localhost:10888`), open `billing-sweep` and **Trigger Run** (or `client.Crons().Create` a `* * * * *` cron temporarily).

- [ ] **Step 4: Verify the fan-out in the UI.** Confirm: `billing-sweep` succeeds → one `org-billing` run per org (filter by `orgId` metadata) → one `billing-cycle-runner` per due sub (filter by `subscriptionId`) → each `billing-cycle-runner` **completes** (not 5-min-reaped, not immortal).

- [ ] **Step 5: Verify state advanced.** The charged subscription's `cycles_processed` incremented and `renews_at` moved forward; a *not-due* subscription was untouched; a *paused* one was not selected.

- [ ] **Step 6: Verify idempotency.** Trigger `billing-sweep` again in the same hour → no second charge for the same cycle (run-key dedup), `cycles_processed` unchanged.

- [ ] **Step 7: Verify reminders (per-tenant setting).** `PUT /api/billing/reminder-config` with `{"enabled":true,"offsets":["168h","24h"]}` for the org. Set an active subscription's `renews_at` to ~12 hours out (inside the 24h stage, past the 168h stage). Trigger `billing-sweep` → confirm one `send-renewal-reminder` run for the `24h` stage (filter by `reminderOffset` metadata) and that re-triggering in the same hour does **not** send again (per-`(cycle,offset)` dedup). Then `PUT {"enabled":false,...}` → confirm no reminder runs spawn. Also verify an org with **no** setting falls back to `DefaultReminderConfig()` (7-day reminder).

- [ ] **Step 8: Record results** in a short note under this task and stop (no commit needed).

---

## Task 9: Temporal reminder parity (engine-parity rule)

Make the Temporal `SubscriptionWorkflow` honor the same per-tenant `ReminderConfig`, resolved **once per billing cycle** (the agreed consistency model). Temporal keeps its durable-runner billing model (valid there — it already uses `ContinueAsNew`); only reminder *scheduling* changes. The runner reaches the shared resolver through an **activity** (workflow code must stay deterministic).

**Files:**
- Modify: `internal/adapter/temporal/activities/order_activities.go` (add resolver field + `ResolveReminderConfig` activity)
- Modify: `internal/adapter/temporal/workflows/subscription_workflow.go` (per-offset reminders, config resolved each cycle)
- Modify: `internal/adapter/temporal/workflows/keys.go` (`ReminderWorkflowID` → per-(cycle,offset))
- Modify: `internal/config/app.go` (inject `reminderConfigService` into `NewOrderActivities`)

- [ ] **Step 1: Add the resolver to `OrderActivities`**

In `internal/adapter/temporal/activities/order_activities.go`, add a field `reminderResolver port.ReminderConfigResolver` to the struct, add it as a parameter to `NewOrderActivities(...)` (set it in the returned struct), and add the activity:

```go
// ResolveReminderConfig exposes the per-tenant reminder policy to the durable
// SubscriptionWorkflow. Workflows can't do I/O directly, so this activity wraps
// the shared resolver — the same one the Hatchet sweep uses.
func (a *OrderActivities) ResolveReminderConfig(ctx context.Context, orgId string) (domain.ReminderConfig, error) {
	return a.reminderResolver.ResolveReminderConfig(ctx, orgId)
}
```

Confirm `domain` and `port` are imported in that file.

- [ ] **Step 2: Make `ReminderWorkflowID` per-(cycle, offset)**

In `internal/adapter/temporal/workflows/keys.go`, replace `ReminderWorkflowID` (currently keyed by date) so two same-day offset stages don't collide — mirror Hatchet's `ReminderStageRunKey`:

```go
// ReminderWorkflowID de-duplicates a reminder to once per (sub, cycle, offset stage).
func ReminderWorkflowID(orgId, subscriptionId string, cycle int, offset time.Duration) string {
	return fmt.Sprintf("reminder_%s_%s_%d_%s", orgId, subscriptionId, cycle, offset.String())
}
```

Run `grep -rn "ReminderWorkflowID(" internal/adapter/temporal/` to find callers — only the subscription workflow should call it; update that call (Step 3).

- [ ] **Step 3: Schedule per-offset reminders in `SubscriptionWorkflow`**

In `internal/adapter/temporal/workflows/subscription_workflow.go`, replace the single fire-and-forget reminder block (the `reminderAt := next.Add(-1 * time.Minute)` section, ~lines 105-114) with a config-driven loop:

```go
		// Reminders — resolve the per-tenant config ONCE per cycle (changes apply
		// next cycle), then schedule one detached child per offset stage.
		var reminderCfg domain.ReminderConfig
		_ = temporal.ExecuteActivity(actCtx, act.ResolveReminderConfig, sub.OrgId).Get(ctx, &reminderCfg)
		if reminderCfg.Enabled {
			for _, offset := range reminderCfg.Offsets {
				reminderAt := next.Add(-offset)
				if reminderAt.Before(temporal.Now(ctx)) {
					continue // this stage's lead time already passed for this cycle
				}
				reminderCtx := temporal.WithChildOptions(ctx, temporal.ChildWorkflowOptions{
					WorkflowID:            ReminderWorkflowID(sub.OrgId, sub.Id, sub.CyclesProcessed, offset),
					ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
					WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				})
				_ = temporal.ExecuteChildWorkflow(reminderCtx, SubscriptionChargeReminder, ReminderInput{
					Subscription: sub,
					ReminderAt:   reminderAt,
				}).GetChildWorkflowExecution().Get(ctx, nil)
			}
		}
```

> Verify: `actCtx` — use the workflow's existing activity-options context (the one used for other `ExecuteActivity` calls in this file); if none is in scope at this point, create it with `temporal.WithActivityOptions(ctx, ...)` mirroring a sibling workflow. Confirm `domain` and the `enums` package are already imported (they are — the old block used `enums`). `SubscriptionChargeReminder` and `ReminderInput` are unchanged (the Temporal reminder workflow stays; only its scheduling is now config-driven).

- [ ] **Step 4: Inject the resolver in app.go**

In `internal/config/app.go`, add `reminderConfigService` (built in Task 4E) to the `activities.NewOrderActivities(...)` argument list. Confirm `NewTemporalEngine` still receives the constructed `orderActivities`.

- [ ] **Step 5: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 6: Verify (build + replay/unit; e2e needs a Temporal server)**

Run: `go test ./internal/adapter/temporal/...`
Expected: PASS. The local stack has no Temporal server (per CLAUDE.md), so full e2e requires one at `TEMPORAL_HOST`. If available: run with `WORKFLOW_ENGINE=temporal`, set a sub's `renews_at` ~12h out with a `24h,168h` reminder config, and confirm the runner spawns the `24h`-stage `SubscriptionChargeReminder` child once; editing the config mid-cycle does **not** change the current cycle's reminders (applies next cycle).

- [ ] **Step 7: Commit**

```bash
git add internal/adapter/temporal/ internal/config/app.go
git commit -m "feat(temporal): per-tenant reminder config parity (resolve per cycle)"
```

---

## Self-Review checklist (completed during planning)

- **Spec coverage:** cron entrypoint (Task 5), per-org fan-out (Task 5), due query (Task 2), upcoming-renewals query (Task 2B), org enumeration (Task 1), one-bounded-cycle task (Task 4), **per-tenant reminders** (domain+repo Task 4B, resolver service+port Task 4C, workflow Task 4D, tenant endpoint Task 4E, key Task 3, fan-out Task 5, wiring Task 6, e2e Task 8 Step 7), wiring + cron registration (Task 6), retire immortal runner (Tasks 6-7), idempotency guard (Task 0), **Temporal reminder parity** (Task 9), e2e proof (Task 8). ✅
- **Engine-parity check:** reminders honored on **both** engines — Hatchet (sweep, Task 5) + Temporal (runner, Task 9), sharing one `ReminderConfigService`. Billing-trigger divergence is the one documented exception (Engine scope section + CLAUDE.md). ✅
- **Replay-safety:** `billing-cycle-runner` is durable + evictable → replay-from-top; Task 0 guards the handlers, charge is key-idempotent. ✅
- **Retention-safety:** every spawned task (`org-billing`, `billing-cycle-runner`, `billing-cycle`) completes → its birth-date partition drops cleanly; no immortal task remains. ✅
- **Type consistency:** `FindDueForBilling(ctx, orgId, now)`, `ListIds(ctx)`, `OrgBillingInput{OrgId}`, `OrgBillingRunKey(orgId, bucket)`, `BillingRunKey(org,sub,cycle)` used consistently across tasks. ✅
- **Known verification points flagged inline:** `OrgScope`, `domain.BillingInterval*` constant names, helper-function locations, `hatchet.Context`/`NewStandaloneTask` signatures, nulltime column names — each task says "verify" where the surrounding code must be confirmed.

## Out-of-scope follow-ups (track separately)

0. **Cedar authz on `PUT /api/billing/reminder-config`** — the endpoint ships authenticated + org-scoped but with **no fine-grained authz** (any authenticated org member can change the reminder config), matching the `OrgHandler` precedent, because `policy.cedar` has no reminder/billing-config action. Hardening follow-up: add a `BillingConfig`/`Reminder` action to `port.Action` + `policy.cedar` and gate the handler (ideally owner/admin), for parity with the dunning/psp config handlers.
1. **Per-plan / per-segment reminder scoping** — this pass resolves reminder config per **org** (from the `settings` table via `ReminderConfigService`). If merchants need different reminder schedules per plan, tier, or customer segment, extend the resolver with a scope/priority lookup like `DunningConfigScope` (the resolver in `org-billing` is the single extension point). Per-org config + disable is already done.
2. **`dunning-runner` eviction** — bounded so retention-safe, but still needs `WithEvictionPolicy` + execution-timeout > TTL so its multi-day progressive waits aren't 5-min-reaped (see timeout doc Fix option 3).
3. **Lago-grade calendar/anniversary date math + multi-currency grouping** — current `GetNextChargeDate` is simpler; revisit if proration/timezone correctness is needed.
4. **Temporal parity** — Temporal keeps the durable runner (no retention-partition issue). Decide whether to converge both engines on the cron model later.
