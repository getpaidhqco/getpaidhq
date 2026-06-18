# Invoice status: Stripe-aligned semantics (uncollectible, no invoice-level unpaid)

**Date:** 2026-06-18
**Repo:** `gphq-server` (branch `worktree-invoice-uncollectible`, off `origin/main`)

## Goal

Make the **invoice** status model mirror Stripe and stop conflating "a charge attempt
failed" with "we gave up collecting":

- Invoice statuses become **`draft`, `open`, `paid`, `uncollectible`, `void`**.
- **Remove `unpaid` from the invoice** entirely. `unpaid` lives on the **subscription**
  (it already does: `SubscriptionStatusUnpaid`) and stays there.
- An invoice is **`open` until it is `paid` or `uncollectible`** — a failed charge with
  retries remaining no longer flips the invoice; it stays `open` while dunning runs.
- **`uncollectible`** is the new terminal "given up collecting" state.

## Current behaviour (the problem)

- Enum: `InvoiceStatus AS ENUM ('draft','open','paid','unpaid','void')`
  (`internal/core/domain/invoice.go`; Postgres type in `schemas/app/migrations/00001_baseline.sql`).
- `open` and `void` are **declared but never assigned**. `draft` is set at build.
- The charge-failure handler (`internal/core/service/subscription.go`) calls
  `MarkUnpaid` **unconditionally on every failed attempt**, before the retry policy is
  even evaluated. So a transient failure (more retries to come) marks the invoice
  `unpaid`, and a later successful retry flips it back to `paid`. There is no terminal
  "given up" invoice state distinct from a transient failure.
- Subscription lifecycle is already correct: `active → past_due → unpaid | cancelled`
  (set only when retries are exhausted, `nextRetryDate.IsZero()`).

## Target state machine (invoice)

```
build ──────────────────────────────▶ draft
first charge attempt for the cycle ─▶ open
charge succeeds ────────────────────▶ paid           (terminal)
charge fails, retries remain ───────▶ open  (no change; subscription → past_due, dunning runs)
retries exhausted, policy ends collection (mark_unpaid | cancel) ─▶ uncollectible (terminal)
retries exhausted, policy = past_due (keep trying) ─▶ open  (no change)
voluntary/admin cancel, per merchant choice (default uncollectible) ─▶ uncollectible | void | open
```

The **subscription** side is unchanged and remains the source of truth for `unpaid`:
`active → past_due` (failure, retries remain) → `unpaid` (exhausted + `mark_unpaid`) or
`cancelled` (exhausted + `cancel`).

### How `cancel` resolves the invoice

There are two cancel sources, and they're treated differently:

| Cancel source | Invoice outcome |
| --- | --- |
| **Dunning exhausted** (`FailureActionCancel`) — the system gave up | **uncollectible** (automatic) |
| **Voluntary / admin cancel** — merchant-initiated | **merchant's choice** (default `uncollectible`) |

Voluntary cancel is user-driven, so the outcome is a **parameter** on the cancel command
(see below), not a hardcoded rule.

### What `CancelSubscription` actually does (corrects an earlier assumption)

`SubscriptionService.CancelSubscription` flips `status → cancelled` **immediately** and sets
`CancelAt = RenewsAt` (access runs to the end of the already-paid period; it simply won't
renew). Billing is **in advance**, so at a normal voluntary cancel the current cycle's
invoice is already `paid` and the next cycle's invoice does not exist yet — **there is
nothing to void or write off.** The only time a non-terminal invoice exists at voluntary
cancel is when the subscription is **`past_due`** (dunning in flight, a real failed-collection
`open` invoice). That invoice is what the merchant's choice applies to.

## Components & changes

### Domain — `internal/core/domain/invoice.go`
- Replace `InvoiceStatusUnpaid` with `InvoiceStatusUncollectible = "uncollectible"`.
  Final set: `draft, open, paid, uncollectible, void`. Update the doc comments.
- Add intention-revealing transition guards on the `Invoice` aggregate (small, testable):
  - `MarkOpen()` (draft → open), `MarkPaid()` (open → paid),
    `MarkUncollectible()` (open → uncollectible), `Void()` (draft|open → void).
  - Each rejects illegal source states (e.g. can't pay a `void`/`uncollectible` invoice),
    returning a sentinel error. Keeps the state machine in one place.

### Service — `internal/core/service/invoice.go`
- Add `MarkOpen(ctx, orgId, invoiceId)`.
- Keep `MarkSettled` → `paid`.
- **Remove `MarkUnpaid`.** Add `MarkUncollectible(ctx, orgId, invoiceId)`.
- Add `Void(ctx, orgId, invoiceId)`.
- All go through the domain guards above (not a raw `setStatus`).

### Charge flow — `internal/core/service/subscription.go`
- When the **first charge attempt** for a cycle begins, transition the invoice
  `draft → open` (`MarkOpen`).
- In `HandleSubscriptionChargeFailure`: **delete the unconditional `MarkUnpaid` call.**
  - Retries remain → leave the invoice `open` (only the subscription moves to `past_due`).
  - In the exhaustion branch (`nextRetryDate.IsZero()`): for `FailureActionMarkUnpaid`
    and `FailureActionCancel`, call `MarkUncollectible`. For `FailureActionLeavePastDue`,
    leave the invoice `open`.
- Success path keeps calling `MarkSettled` (`open → paid`).

### Cancel path — `port.CancelSubscriptionInput`, `subscription.go`, HTTP handler
- Add `OutstandingInvoice` to `CancelSubscriptionInput` (today `{OrgId, Id, Reason}`): an
  enum `uncollectible` (default) | `void` | `keep`, applied to the subscription's
  non-terminal (`draft`/`open`) invoice for the current cycle at voluntary-cancel time:
  - `uncollectible` → `MarkUncollectible` (default — a cancelled sub has no dunning left to
    ever resolve an `open` invoice, so writing it off is the honest state)
  - `void` → `Void` (forgive)
  - `keep` → leave as-is
  An empty value defaults to `uncollectible`.
- Expose it as an optional `outstanding_invoice` field on the cancel HTTP request
  (`subscription_handler.go:134`); omitted → `uncollectible`.
- Dunning-exhaustion cancel (`FailureActionCancel`) is unaffected by this parameter — it
  always sets `uncollectible` in the charge-failure exhaustion branch.

### Database — Goose migration `schemas/app/migrations/00002_invoice_uncollectible.sql`
Postgres cannot drop an enum value, so swap the type:
```sql
-- +goose Up
ALTER TYPE "InvoiceStatus" RENAME TO "InvoiceStatus_old";
CREATE TYPE "InvoiceStatus" AS ENUM ('draft','open','paid','uncollectible','void');
ALTER TABLE "invoices" ALTER COLUMN "status" TYPE "InvoiceStatus"
  USING (CASE "status"::text WHEN 'unpaid' THEN 'open' ELSE "status"::text END)::"InvoiceStatus";
DROP TYPE "InvoiceStatus_old";
-- +goose Down  (reverse: re-add 'unpaid', map 'uncollectible' -> 'unpaid')
```
Existing `unpaid` rows map to **`open`** (decided: treat as still-collectible; the
platform is pre-launch/local-only, so there are likely zero such rows). Verify the
goose Up applies cleanly on a scratch DB and shows zero drift vs the updated domain.

### Contract propagation (downstream)
The status is exposed via the invoice HTTP handler/DTO (`internal/adapter/http/invoice_handler*.go`)
→ `openapi.json`. After the server change: re-export `openapi.json`
(`go run ./cmd/openapi-export`), then update the SDK and the web invoice-status display
(remove `unpaid`, add `uncollectible`). Web/SDK changes are tracked as downstream
follow-on, not in this server plan.

## Testing
- **Domain:** table-driven tests for each transition guard, incl. rejected illegal
  transitions.
- **Service:** `MarkOpen`/`MarkUncollectible`/`Void` happy + illegal-source paths.
- **Integration (`internal/adapter/postgres`, build tag `integration`):** drive the
  charge-failure flow end-to-end and assert: transient failure leaves invoice `open`
  (subscription `past_due`); exhaustion + `mark_unpaid`/`cancel` → `uncollectible`;
  exhaustion + `past_due` → still `open`; success → `paid`. For voluntary cancel of a
  `past_due` subscription, assert the `OutstandingInvoice` choice: default/`uncollectible`
  → `uncollectible`, `void` → `void`, `keep` → unchanged `open`.
- **Migration:** apply `00002` to a scratch DB, confirm the enum is
  `{draft,open,paid,uncollectible,void}` and an `unpaid` row migrates to `open`.
- Update/replace existing tests that asserted invoice `unpaid`.

## Out of scope
- SDK and web changes (downstream of the regenerated `openapi.json`).
- Any change to the subscription state machine (`unpaid` stays on the subscription).
- Reporting-schema or usage-schema changes (unaffected).

## Definition of done
1. Invoice enum/domain/service no longer contain `unpaid`; `uncollectible` present.
2. Charge-failure flow: invoice stays `open` through dunning; `uncollectible` only on
   exhausted-and-collection-ended; voluntary cancel applies the `OutstandingInvoice`
   choice to a `past_due` sub's open invoice (default `uncollectible`).
3. Goose `00002` applies cleanly; `unpaid` rows → `open`; zero drift.
4. `go build ./...`, `go vet ./...`, unit + postgres-integration suites pass.
5. `openapi.json` re-exported reflecting the new enum.
