# Engine parity & the subscription lifecycle

> Private/internal. Read alongside [subscriptions-on-hatchet.md](subscriptions-on-hatchet.md)
> and [durable-runner-timeouts.md](durable-runner-timeouts.md). This is the load-bearing mental
> model for *why* the two engines look different and *where the first charge happens*.

## Engine parity means same **behaviour**, not same **implementation**

`WORKFLOW_ENGINE=hatchet|temporal` selects between two adapters that implement the billing/dunning/
reminder surface **deliberately differently**. Parity is about **observable outcomes** — a
subscription bills on schedule, reminders fire once per stage per cycle, dunning escalates, the same
state lands in the DB — **not** identical implementations.

The two models are opposite on purpose, because the engines' durability primitives are opposite:

| | Temporal | Hatchet |
| --- | --- | --- |
| Core model | **one long-lived durable actor per subscription** (`SubscriptionWorkflow` + `ContinueAsNew`) | **cron + per-org fan-out into short, bounded tasks** |
| Why this model | Temporal's timer/history model supports workflows that live (sleep) indefinitely | Hatchet GC's the durable event log **by creation-date partition**, so an immortal task is reaped mid-flight (see [durable-runner-timeouts.md](durable-runner-timeouts.md)) |
| Renewal "timer" | `workflow.Await` durable sleep until `RenewsAt` | hourly cron `billing-sweep` selects `renews_at <= now` |
| Reaches shared services via | **activities** (workflow code must stay deterministic) | steps + run-key-idempotent spawns |

### The rule this implies (also in CLAUDE.md)

A **behaviour** change to billing/dunning/reminders must land on **both** adapters, so switching
`WORKFLOW_ENGINE` never changes outcomes. But **how** each adapter expresses that behaviour can and
does differ. "Implement on both" ≠ "copy the same code into both" — it means "produce the same
result on both, each in its engine's idiom." Put new *logic* in `core/` so both literally share it;
only the *orchestration* is per-adapter.

---

## The subscription lifecycle: first charge → renewals

### Shared prelude (both engines)

A PSP payment-confirmed webhook starts the **durable `payment-success` workflow**
(`webhook.go:121` → `StartWorkflow(PaymentSuccess)`):

1. `complete-order` → `OrderService.CompleteOrder` → `subscription.SetActive(payment)`
   (`order.go:426`, `subscription.go:298`). **This is where the FIRST charge is recorded:**
   `TotalRevenue`/`LastCharge` set, `CyclesProcessed++` (cycle 1 = the checkout payment), and
   `RenewsAt = CalculateNextBillingDate()` — the **next** cycle, in the future.
2. `get-subscriptions` loads the order's subscriptions.
3. Hand-off to the renewal mechanism — **this is where the engines diverge.**

> **"…or logged if done outside the system":** `SetActive` only *records* the payment (revenue,
> last-charge, cycle) — it does not re-charge. So a first payment collected externally is passed in
> as a `Payment` and logged, then the subscription is set up for future cycles. Same on both engines.

### Where the first charge lives — by case, by engine

| | Temporal | Hatchet |
| --- | --- | --- |
| **First charge (paid at checkout — common)** | recorded by `SetActive` in `payment-success`; **not** the runner | **same** — recorded by `SetActive` in `payment-success` |
| **Activation hand-off** | `payment-success` starts the durable `SubscriptionWorkflow` (per-sub runner) as a detached child (`payment_success.go`) | `payment-success` → `StartSubscriptionWorkflow` is a **no-op** (the cron sweep drives renewals) |
| **Renewals** | runner loop: `Await` until `RenewsAt` → `BillingCycleWorkflow` → `HandleChargeResult`; `ContinueAsNew` to stay alive (`subscription_workflow.go`) | hourly `billing-sweep` → `org-billing` → `billing-cycle-runner` when `renews_at <= now` |
| **Immediate first charge** (no upfront payment, `RenewsAt <= now`) | the runner's **first loop iteration** charges (`next <= now` → `wait` clamps to ~1s → `BillingCycleWorkflow`) | a **direct `billing-cycle-runner` spawn at activation** (intended) — the analog of Temporal's first iteration; **today** it falls to the sweep (≤1h late) |

### The key asymmetry

- **Temporal** handles "first charge if due" *and* every renewal uniformly inside **one** long-lived
  actor. The first charge is never a separate thing — it's either the checkout payment (recorded at
  activation) or the runner's **first iteration**. Durability = a persisted, replayable, timer-driven
  workflow kept alive by `ContinueAsNew`.
- **Hatchet** cannot keep an immortal actor (retention reaps it), so it splits the responsibilities:
  **renewals → the cron sweep**; the **immediate first charge → a direct spawn of
  `billing-cycle-runner` at activation**. That spawn is the same bounded, durable, per-cycle unit the
  sweep uses, keyed `BillingRunKey(org, sub, cycle)` — so the activation-spawn and any sweep-spawn
  **dedup**, and the sweep is a free backstop.

  > **Don't invent a separate `first-charge` workflow** — `billing-cycle-runner` already *is* the
  > durable per-cycle charge unit, and `HandleSubscriptionChargeSuccess` already sets
  > `Status = Active`. The only real difference between a first charge and a renewal is **period
  > initialization**: `SetActive` seeds `CurrentPeriodStart` from `StartDate`, whereas
  > `HandleSubscriptionChargeSuccess` rolls `CurrentPeriodStart = CurrentPeriodEnd` (unset → wrong
  > boundaries on cycle 1). Fix in the *handler* (init from `StartDate` when `CurrentPeriodEnd` is
  > zero), not in a new workflow. Tracked as a follow-up.

**Net (both engines, same outcome):** the first charge is durable and **outside the renewal
mechanism**; renewals are durable. The *outcome* is identical; the *implementation* is opposite — and
that's the point.
