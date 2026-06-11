# Carry-over meters: flow vs stock

> The low-level mechanic behind `BillableMetric.CarryOver`. For the canonical
> stock use case (per-seat billing) see [`seat-billing/`](./seat-billing/).

There are two fundamentally different shapes of usage:

**FLOW** *(carry_over = false)*: discrete events that happen and are "spent" —
API calls, emails, GB transferred.
→ each period counts only its own events; resets to zero next period.

**STOCK** *(carry_over = true)*: a quantity that persists until changed —
active seats, provisioned VMs, enabled features.
→ the current amount carries forward; you bill what's standing.

That distinction is baked into how you send events and how the data must be
read — which is why it lives on the metric:

1. **The event shape differs.**
   A flow metric gets plain events (`api_call`, count it).
   A stock metric gets add/remove events carrying an operation and an identity
   (`operation: add|remove`, `seat_id`) — or, for merchants who can only report
   totals, level reports (`count: 5`).

   The same raw event stream can't be both — so whether the metric carries over
   determines the contract for how you instrument it. That's intrinsic to the
   metric.

2. **The aggregation boundary differs.**
   The one switch `carry_over` flips internally is `use_from_boundary = !carry_over`:
   - **flow** (`carry_over = false`) → the query applies the period's start
     boundary → only this period's events count → resets each period.
   - **stock** (`carry_over = true`) → the query drops the start boundary → it
     reads back through all prior add/remove history to reconstruct "what's
     currently active" → state carries across periods.

So `carry_over` is really answering: *"to know the value right now, do I count
this period's events, or do I replay the whole history?"* That's a question
about the data, identical no matter which plan or price consumes it.

> Implemented in `UsageService.AggregateForPeriod`: a carry-over meter fetches its
> full event history (`EventStore.ListHistory`) and the standing-level math in
> `internal/core/domain/usage_interval.go` computes the quantity. See
> [`seat-billing/mapping.md`](./seat-billing/mapping.md) and
> [`stock-billing-architecture-impact.md`](./stock-billing-architecture-impact.md).
