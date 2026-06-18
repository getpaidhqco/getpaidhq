# Invoice Status: uncollectible (no invoice-level unpaid) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the invoice-level `unpaid` status with `uncollectible`, keep invoices `open` through dunning, and let cancel choose the outstanding-invoice outcome.

**Architecture:** Move the invoice state machine into guarded methods on the `Invoice` domain aggregate; the service routes every transition through them. The charge flow sets `open` before charging and `uncollectible` only when recovery ends. A Goose `00002` type-swap drops the `unpaid` enum value.

**Tech Stack:** Go 1.26, GORM, Postgres (enum type), Goose, testify, testcontainers (integration).

**Working dir:** `gphq-server` worktree `worktree-invoice-uncollectible` (off `origin/main`). Paths are repo-root-relative.

**Reference spec:** `docs/superpowers/specs/2026-06-18-invoice-status-uncollectible-design.md`

---

## File Structure

**Modify:**
- `internal/core/domain/invoice.go` — enum (`unpaid`→`uncollectible`) + transition guard methods + sentinel error
- `internal/core/service/invoice.go` — `MarkOpen`/`MarkUncollectible`/`Void`; `MarkSettled` via guard; remove `MarkUnpaid`
- `internal/core/service/subscription.go` — `MarkOpen` in `ChargeForBillingPeriod`; rework failure handler
- `internal/core/port/subscription_input.go` — `OutstandingInvoice` on `CancelSubscriptionInput` + enum
- `internal/adapter/http/subscription_handler.go` — read `outstanding_invoice` from the cancel request

**Create:**
- `schemas/app/migrations/00002_invoice_uncollectible.sql` — enum type-swap

**Tests:** `internal/core/domain/invoice_test.go`, `internal/core/service/invoice_test.go`, the postgres integration tests, plus any existing test asserting invoice `unpaid`.

---

## Task 1: Domain enum + transition guards

**Files:** Modify `internal/core/domain/invoice.go`; Test `internal/core/domain/invoice_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/core/domain/invoice_test.go`:

```go
package domain

import (
	"errors"
	"testing"
)

func TestInvoiceTransitions(t *testing.T) {
	cases := []struct {
		name    string
		from    InvoiceStatus
		apply   func(*Invoice) error
		want    InvoiceStatus
		wantErr bool
	}{
		{"draft->open", InvoiceStatusDraft, (*Invoice).MarkOpen, InvoiceStatusOpen, false},
		{"open->open idempotent", InvoiceStatusOpen, (*Invoice).MarkOpen, InvoiceStatusOpen, false},
		{"open->paid", InvoiceStatusOpen, (*Invoice).MarkPaid, InvoiceStatusPaid, false},
		{"open->uncollectible", InvoiceStatusOpen, (*Invoice).MarkUncollectible, InvoiceStatusUncollectible, false},
		{"draft->void", InvoiceStatusDraft, (*Invoice).Void, InvoiceStatusVoid, false},
		{"open->void", InvoiceStatusOpen, (*Invoice).Void, InvoiceStatusVoid, false},
		{"paid->open rejected", InvoiceStatusPaid, (*Invoice).MarkOpen, InvoiceStatusPaid, true},
		{"void->paid rejected", InvoiceStatusVoid, (*Invoice).MarkPaid, InvoiceStatusVoid, true},
		{"uncollectible->void rejected", InvoiceStatusUncollectible, (*Invoice).Void, InvoiceStatusUncollectible, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inv := Invoice{Status: c.from}
			err := c.apply(&inv)
			if c.wantErr {
				if !errors.Is(err, ErrInvalidInvoiceTransition) {
					t.Fatalf("want ErrInvalidInvoiceTransition, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if inv.Status != c.want {
				t.Fatalf("status = %q, want %q", inv.Status, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run, verify it fails to compile**

Run: `go test ./internal/core/domain/ -run TestInvoiceTransitions`
Expected: FAIL — `InvoiceStatusUncollectible`, `ErrInvalidInvoiceTransition`, `MarkOpen`, etc. undefined.

- [ ] **Step 3: Implement enum + guards**

In `internal/core/domain/invoice.go`, change the import line `import "time"` to:

```go
import (
	"errors"
	"time"
)
```

Replace the `const (...)` enum block so `InvoiceStatusUnpaid` becomes `InvoiceStatusUncollectible`:

```go
const (
	InvoiceStatusDraft         InvoiceStatus = "draft"         // built, not yet charged
	InvoiceStatusOpen          InvoiceStatus = "open"          // finalized, a charge is outstanding
	InvoiceStatusPaid          InvoiceStatus = "paid"          // settled by a succeeded Payment
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible" // collection given up (terminal)
	InvoiceStatusVoid          InvoiceStatus = "void"          // cancelled, never collected (terminal)
)

// ErrInvalidInvoiceTransition is returned when a status change is not allowed
// from the invoice's current state.
var ErrInvalidInvoiceTransition = errors.New("invalid invoice status transition")
```

Add the guard methods at the end of the file:

```go
// MarkOpen finalizes a draft invoice for collection. Idempotent from open.
func (inv *Invoice) MarkOpen() error {
	switch inv.Status {
	case InvoiceStatusDraft, InvoiceStatusOpen:
		inv.Status = InvoiceStatusOpen
		return nil
	default:
		return ErrInvalidInvoiceTransition
	}
}

// MarkPaid settles an outstanding invoice.
func (inv *Invoice) MarkPaid() error {
	switch inv.Status {
	case InvoiceStatusDraft, InvoiceStatusOpen:
		inv.Status = InvoiceStatusPaid
		return nil
	default:
		return ErrInvalidInvoiceTransition
	}
}

// MarkUncollectible writes off an outstanding invoice (collection abandoned).
func (inv *Invoice) MarkUncollectible() error {
	switch inv.Status {
	case InvoiceStatusDraft, InvoiceStatusOpen:
		inv.Status = InvoiceStatusUncollectible
		return nil
	default:
		return ErrInvalidInvoiceTransition
	}
}

// Void cancels an invoice that should never be collected.
func (inv *Invoice) Void() error {
	switch inv.Status {
	case InvoiceStatusDraft, InvoiceStatusOpen:
		inv.Status = InvoiceStatusVoid
		return nil
	default:
		return ErrInvalidInvoiceTransition
	}
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/core/domain/ -run TestInvoiceTransitions`
Expected: PASS.

- [ ] **Step 5: Confirm no stale `InvoiceStatusUnpaid` references compile-break elsewhere yet**

Run: `grep -rn "InvoiceStatusUnpaid" internal --include=*.go`
Expected: only `internal/core/service/invoice.go:138` (handled in Task 2). Note them; do not fix here.

- [ ] **Step 6: Commit**

```bash
git add internal/core/domain/invoice.go internal/core/domain/invoice_test.go
git commit -m "feat(domain): invoice uncollectible status + guarded transitions"
```

---

## Task 2: Service transitions through the guards

**Files:** Modify `internal/core/service/invoice.go`; Test `internal/core/service/invoice_test.go`

The current service has `MarkSettled` (→paid), `MarkUnpaid` (→unpaid), and private `setStatus` (FindById → set → Update) at lines 131-149.

- [ ] **Step 1: Write failing test**

Add to `internal/core/service/invoice_test.go` (create if absent, `package service`). Use the existing in-memory invoice repo test pattern if one exists; otherwise this minimal fake:

```go
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type fakeInvoiceRepo struct{ inv domain.Invoice }

func (f *fakeInvoiceRepo) Create(_ context.Context, in domain.Invoice) (domain.Invoice, error) { f.inv = in; return in, nil }
func (f *fakeInvoiceRepo) Update(_ context.Context, in domain.Invoice) (domain.Invoice, error) { f.inv = in; return in, nil }
func (f *fakeInvoiceRepo) FindById(_ context.Context, _, _ string) (domain.Invoice, error)     { return f.inv, nil }
func (f *fakeInvoiceRepo) FindBySubscriptionCycle(_ context.Context, _, _ string, _ int) (domain.Invoice, error) { return f.inv, port.ErrNotFound }
func (f *fakeInvoiceRepo) FindBySubscriptionId(_ context.Context, _, _ string, _ domain.Pagination) ([]domain.Invoice, int, error) { return nil, 0, nil }
func (f *fakeInvoiceRepo) List(_ context.Context, _ string, _ domain.Pagination) ([]domain.Invoice, int, error) { return nil, 0, nil }

func TestInvoiceServiceTransitions(t *testing.T) {
	repo := &fakeInvoiceRepo{inv: domain.Invoice{Id: "inv_1", Status: domain.InvoiceStatusOpen}}
	s := &InvoiceService{invoiceRepository: repo}
	ctx := context.Background()

	got, err := s.MarkUncollectible(ctx, "org", "inv_1")
	require.NoError(t, err)
	require.Equal(t, domain.InvoiceStatusUncollectible, got.Status)

	repo.inv = domain.Invoice{Id: "inv_1", Status: domain.InvoiceStatusPaid}
	_, err = s.MarkUncollectible(ctx, "org", "inv_1")
	require.ErrorIs(t, err, domain.ErrInvalidInvoiceTransition)
}
```

NOTE: confirm the real `port.InvoiceRepository` method set first (`grep -n "InvoiceRepository interface" -A12 internal/core/port/*.go`) and match the fake to it exactly (method names/signatures may differ — adapt the fake, don't change the port).

- [ ] **Step 2: Run, verify fail**

Run: `go test ./internal/core/service/ -run TestInvoiceServiceTransitions`
Expected: FAIL — `MarkUncollectible` undefined.

- [ ] **Step 3: Implement**

In `internal/core/service/invoice.go`, replace the block from `// MarkSettled ...` through the end of `setStatus` (lines ~131-149) with:

```go
// MarkOpen finalizes a draft invoice for collection (draft -> open).
func (s *InvoiceService) MarkOpen(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.transition(ctx, orgId, invoiceId, (*domain.Invoice).MarkOpen)
}

// MarkSettled flips an invoice to paid after a succeeded Payment.
func (s *InvoiceService) MarkSettled(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.transition(ctx, orgId, invoiceId, (*domain.Invoice).MarkPaid)
}

// MarkUncollectible writes off an invoice after recovery is abandoned.
func (s *InvoiceService) MarkUncollectible(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.transition(ctx, orgId, invoiceId, (*domain.Invoice).MarkUncollectible)
}

// Void cancels an invoice that should not be collected.
func (s *InvoiceService) Void(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return s.transition(ctx, orgId, invoiceId, (*domain.Invoice).Void)
}

func (s *InvoiceService) transition(ctx context.Context, orgId, invoiceId string, apply func(*domain.Invoice) error) (domain.Invoice, error) {
	inv, err := s.invoiceRepository.FindById(ctx, orgId, invoiceId)
	if err != nil {
		return domain.Invoice{}, err
	}
	if err := apply(&inv); err != nil {
		return domain.Invoice{}, err
	}
	inv.UpdatedAt = time.Now().UTC()
	return s.invoiceRepository.Update(ctx, inv)
}
```

`MarkUnpaid` is now gone. `time` and `domain` imports remain in use.

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/core/service/ -run TestInvoiceServiceTransitions`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/core/service/invoice.go internal/core/service/invoice_test.go
git commit -m "feat(service): guarded invoice transitions; drop MarkUnpaid"
```

---

## Task 3: Charge flow — open before charge, uncollectible on exhaustion

**Files:** Modify `internal/core/service/subscription.go`

- [ ] **Step 1: Set `open` in `ChargeForBillingPeriod`**

In `ChargeForBillingPeriod`, immediately after the build log line (`s.logger.Infof("ChargeForBillingPeriod [%s] invoice=%s total=%d", ...)`, ~line 725) and before `gw, err := s.gatewayFactory.NewGateway(...)`, insert:

```go
	if _, err := s.invoiceService.MarkOpen(ctx, subscription.OrgId, invoice.Id); err != nil {
		s.logger.Error("Failed to mark invoice open", "err", err.Error())
	}
```

- [ ] **Step 2: Rework the failure handler**

In `HandleSubscriptionChargeFailure`, **delete** these lines (~644-646):

```go
	if _, err := s.invoiceService.MarkUnpaid(ctx, subscription.OrgId, inv.Id); err != nil {
		s.logger.Error("Failed to mark invoice unpaid", "err", err.Error())
	}
```

Then in the exhaustion branch (`if nextRetryDate.IsZero() {`), add the invoice write-off for the two collection-ending actions. Change the branch body to:

```go
	if nextRetryDate.IsZero() {
		s.logger.Debugf("Subscription [%s] has no more retries left", subscription.Id)
		if retryPolicy.FailureAction == domain.FailureActionMarkUnpaid {
			s.logger.Debugf("Marking as unpaid..")
			subscription.Status = domain.SubscriptionStatusUnpaid
		}
		if retryPolicy.FailureAction == domain.FailureActionCancel {
			s.logger.Debugf("Cancelling..")
			subscription.SetCancelled()
		}
		if retryPolicy.FailureAction == domain.FailureActionMarkUnpaid || retryPolicy.FailureAction == domain.FailureActionCancel {
			if _, err := s.invoiceService.MarkUncollectible(ctx, subscription.OrgId, inv.Id); err != nil {
				s.logger.Error("Failed to mark invoice uncollectible", "err", err.Error())
			}
		}
	} else {
		s.logger.Debugf("Subscription [%s] next retry date [%s]", subscription.Id, nextRetryDate)
		subscription.Status = domain.SubscriptionStatusPastDue
		subscription.NextRetryAt = nextRetryDate
		subscription.Retries++
	}
```

(`FailureActionLeavePastDue` falls through: invoice stays `open`.)

- [ ] **Step 3: Compile**

Run: `go build ./...`
Expected: clean — no remaining `MarkUnpaid` reference.

- [ ] **Step 4: Run service + domain unit tests**

Run: `go test ./internal/core/...`
Expected: PASS (fix any unit test that asserted the old invoice `unpaid`; those move to Task 6's integration assertions or are updated to `open`/`uncollectible`).

- [ ] **Step 5: Commit**

```bash
git add internal/core/service/subscription.go
git commit -m "feat(billing): invoice open at charge, uncollectible on exhausted recovery"
```

---

## Task 4: Cancel chooses the outstanding-invoice outcome

**Files:** Modify `internal/core/port/subscription_input.go`, `internal/core/service/subscription.go`, `internal/adapter/http/subscription_handler.go`

- [ ] **Step 1: Add the action enum + input field**

In `internal/core/port/subscription_input.go`, above `CancelSubscriptionInput`, add:

```go
// OutstandingInvoiceAction decides what happens to a still-open invoice when a
// subscription is voluntarily cancelled. Empty defaults to uncollectible.
type OutstandingInvoiceAction string

const (
	OutstandingInvoiceUncollectible OutstandingInvoiceAction = "uncollectible"
	OutstandingInvoiceVoid          OutstandingInvoiceAction = "void"
	OutstandingInvoiceKeep          OutstandingInvoiceAction = "keep"
)
```

Add the field to the struct:

```go
type CancelSubscriptionInput struct {
	OrgId              string
	Id                 string
	Reason             string
	OutstandingInvoice OutstandingInvoiceAction // empty => uncollectible
}
```

- [ ] **Step 2: Apply it in `CancelSubscription`**

In `internal/core/service/subscription.go`, at the end of `CancelSubscription` just before `return subscription, nil` (after the tx commits, ~line 388), add:

```go
	s.applyOutstandingInvoiceAction(ctx, subscription, input.OutstandingInvoice)
```

And add this method (next to `CancelSubscription`):

```go
// applyOutstandingInvoiceAction resolves the subscription's current-cycle
// invoice when a voluntary cancel leaves one open (only happens if the sub was
// past_due — billing is in advance, so an active sub's invoice is already paid).
func (s *SubscriptionService) applyOutstandingInvoiceAction(ctx context.Context, sub domain.Subscription, action port.OutstandingInvoiceAction) {
	if action == port.OutstandingInvoiceKeep {
		return
	}
	inv, err := s.invoiceService.invoiceRepository.FindBySubscriptionCycle(ctx, sub.OrgId, sub.Id, sub.CyclesProcessed)
	if err != nil {
		return // no current-cycle invoice (ErrNotFound) — nothing to do
	}
	if inv.Status != domain.InvoiceStatusOpen && inv.Status != domain.InvoiceStatusDraft {
		return // already terminal (paid/uncollectible/void)
	}
	switch action {
	case port.OutstandingInvoiceVoid:
		_, err = s.invoiceService.Void(ctx, sub.OrgId, inv.Id)
	default: // "" or uncollectible
		_, err = s.invoiceService.MarkUncollectible(ctx, sub.OrgId, inv.Id)
	}
	if err != nil {
		s.logger.Error("Failed to apply outstanding-invoice action on cancel", "err", err.Error(), "action", string(action))
	}
}
```

NOTE: if `InvoiceService.invoiceRepository` is unexported and not reachable from `SubscriptionService`, instead add a thin `InvoiceService.FindCurrentCycle(ctx, orgId, subId, cycle)` method and call that. Verify reachability first (`grep -n "invoiceRepository" internal/core/service/invoice.go`) and pick the reachable form.

- [ ] **Step 3: Wire the HTTP field**

In `internal/adapter/http/subscription_handler.go` around line 134 where `port.CancelSubscriptionInput{...}` is built: add `OutstandingInvoice: port.OutstandingInvoiceAction(<request field>)`. Read the cancel request struct first (`grep -n "Cancel" internal/adapter/http/subscription_handler.go` and the DTO it binds); add an optional `OutstandingInvoice string \`json:"outstanding_invoice"\`` to that request DTO (omitempty), defaulting to empty (→ uncollectible). If cancel takes no body today, add a minimal request body struct bound via fuego.

- [ ] **Step 4: Build + vet**

Run: `go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/core/port/subscription_input.go internal/core/service/subscription.go internal/adapter/http/subscription_handler.go
git commit -m "feat(subscription): cancel chooses outstanding-invoice action (default uncollectible)"
```

---

## Task 5: Goose migration — drop `unpaid`, add `uncollectible`

**Files:** Create `schemas/app/migrations/00002_invoice_uncollectible.sql`

- [ ] **Step 1: Write the migration**

Create `schemas/app/migrations/00002_invoice_uncollectible.sql`:

```sql
-- +goose Up
-- Postgres can't drop an enum value, so swap the type. Existing 'unpaid'
-- invoices map to 'open' (still collectible).
ALTER TYPE "InvoiceStatus" RENAME TO "InvoiceStatus_old";
CREATE TYPE "InvoiceStatus" AS ENUM ('draft', 'open', 'paid', 'uncollectible', 'void');
ALTER TABLE "invoices" ALTER COLUMN "status" TYPE "InvoiceStatus"
  USING (CASE "status"::text WHEN 'unpaid' THEN 'open' ELSE "status"::text END)::"InvoiceStatus";
DROP TYPE "InvoiceStatus_old";

-- +goose Down
ALTER TYPE "InvoiceStatus" RENAME TO "InvoiceStatus_new";
CREATE TYPE "InvoiceStatus" AS ENUM ('draft', 'open', 'paid', 'unpaid', 'void');
ALTER TABLE "invoices" ALTER COLUMN "status" TYPE "InvoiceStatus"
  USING (CASE "status"::text WHEN 'uncollectible' THEN 'unpaid' ELSE "status"::text END)::"InvoiceStatus";
DROP TYPE "InvoiceStatus_new";
```

- [ ] **Step 2: Verify up + down on a scratch DB**

Run (local compose Postgres up via `make up`):

```bash
PGBASE="postgres://getpaidhq:getpaidhq@localhost:10432"
psql "$PGBASE/getpaidhq?sslmode=disable" -c "DROP DATABASE IF EXISTS mig_check"
psql "$PGBASE/getpaidhq?sslmode=disable" -c "CREATE DATABASE mig_check"
GOOSE_DRIVER=postgres GOOSE_DBSTRING="$PGBASE/mig_check?sslmode=disable" GOOSE_MIGRATION_DIR=schemas/app/migrations go tool goose up
psql "$PGBASE/mig_check?sslmode=disable" -c "INSERT INTO invoices (org_id,id,subscription_id,customer_id,order_id,status,currency,subtotal,discount_total,total,cycle,period_start,period_end,created_at,updated_at) VALUES ('o','i','s','c','ord','unpaid','USD',0,0,0,0,now(),now(),now(),now())" 2>&1 | tail -1
```
Expected: `goose up` OK; the manual INSERT with `'unpaid'` **fails** (`invalid input value for enum "InvoiceStatus": "unpaid"`) — proving `unpaid` is gone from the new type. Then:
```bash
psql "$PGBASE/mig_check?sslmode=disable" -c "SELECT enumlabel FROM pg_enum JOIN pg_type t ON t.oid=enumtypid WHERE t.typname='InvoiceStatus' ORDER BY enumsortorder"
```
Expected labels: `draft, open, paid, uncollectible, void`. Clean up: `psql "$PGBASE/getpaidhq?sslmode=disable" -c "DROP DATABASE IF EXISTS mig_check"`.

- [ ] **Step 3: Commit**

```bash
git add schemas/app/migrations/00002_invoice_uncollectible.sql
git commit -m "feat(db): migration 00002 - drop invoice 'unpaid' enum, add 'uncollectible'"
```

---

## Task 6: Integration tests, contract, full verification

**Files:** Modify postgres integration tests; re-export `openapi.json`; fix stale `unpaid` test assertions

- [ ] **Step 1: Find and update stale `unpaid` invoice assertions**

Run: `grep -rn "InvoiceStatusUnpaid\|\"unpaid\"" internal --include=*.go`
For any **invoice** assertion (not subscription `unpaid`), change the expectation to the new behavior: transient failure → `InvoiceStatusOpen`; exhausted+mark_unpaid/cancel → `InvoiceStatusUncollectible`. Leave `SubscriptionStatusUnpaid` assertions untouched.

- [ ] **Step 2: Add/extend the charge-failure integration test**

In the postgres integration suite (`internal/adapter/postgres`, build tag `integration`), add assertions driving the failure flow via the services (follow the existing billing-e2e test setup):
- one failed charge with retries remaining → invoice `open`, subscription `past_due`;
- exhausted with `FailureActionMarkUnpaid` → invoice `uncollectible`, subscription `unpaid`;
- exhausted with `FailureActionLeavePastDue` → invoice stays `open`;
- success → invoice `paid`;
- voluntary `CancelSubscription` on a `past_due` sub with `OutstandingInvoice` unset → invoice `uncollectible`; with `void` → `void`; with `keep` → `open`.

- [ ] **Step 3: Run integration tests**

Run: `go test -tags=integration -count=1 ./internal/adapter/postgres/...`
Expected: PASS (the harness applies `schemas/app/migrations`, now including `00002`).

- [ ] **Step 4: Re-export the OpenAPI spec**

Run: `go run ./cmd/openapi-export` (confirm the exact command via `grep -rn "openapi-export\|openapi.json" Makefile package.json README.md`).
Expected: `openapi.json` updates — invoice `status` enum now lists `uncollectible`, not `unpaid`.

- [ ] **Step 5: Full build/vet/test sweep**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all clean/pass.

- [ ] **Step 6: Commit**

```bash
git add -A internal openapi.json
git commit -m "test(billing): invoice open/uncollectible/void flow; re-export openapi"
```

---

## Self-Review Notes

- **Spec coverage:** enum+guards (T1), service (T2), charge flow open/uncollectible (T3), cancel param (T4), Goose 00002 + unpaid→open (T5), integration+contract+stale-test fixes (T6). All spec sections mapped. SDK/web are spec-declared downstream (not in this server plan).
- **Ordering:** domain (T1) → service (T2) → callers (T3, T4) → migration (T5) → integration/contract (T6). Migration before integration so the test harness picks up `00002`.
- **Type consistency:** `MarkOpen/MarkPaid/MarkUncollectible/Void` (domain) ↔ `MarkOpen/MarkSettled/MarkUncollectible/Void` (service; `MarkSettled` wraps `MarkPaid`, preserving its existing callers). `OutstandingInvoiceAction` enum + `CancelSubscriptionInput.OutstandingInvoice` used consistently in T4.
- **Verify-points the executor must confirm against real code:** the `port.InvoiceRepository` method set (T2 fake), `InvoiceService.invoiceRepository` reachability from `SubscriptionService` (T4 — else add `FindCurrentCycle`), the cancel request DTO shape (T4 HTTP), and the openapi-export command (T6).
