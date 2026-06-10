# Use case B — Time-weighted seat billing (prorated)

> **[← Index](./README.md)** · Siblings: [A — Full-period](./use-case-a-full-period.md) · [C — Hybrid](./use-case-c-hybrid.md) · [How we bill it →](./mapping.md#use-case-b)

## Definition

A seat is charged **only for the portion of the period it actually existed**. A
seat held for the entire period costs a full seat; a seat held for half the
period costs half a seat. The bill is the **sum of each seat's share of time**,
so the customer pays precisely for the time each seat was held — which maps to a
`weighted_sum` aggregation.

The business is saying: **"You pay for what you held, for as long as you held
it."** This is the fair, usage-true model: a person who joins on the 16th costs
roughly half, a person who leaves on the 21st stops accruing cost the day they
leave. It removes the friction of *"we got charged a full month for someone who
was here three days,"* at the cost of producing bills that are fractional and
harder to predict.

Each seat is charged for **its share of the period** — its slice of the days.

```
alice   ████████████████████████  whole month   →  1.00
carol   ████████████████████████  whole month   →  1.00
bob     ████████████████░░░░░░░░  ~20 of 30 days →  0.67
dave    ░░░░░░░░░░░░████████████  ~15 of 30 days →  0.50
                                                   ─────
                                      charged for   3.17 seats
```

The question it answers: *"For how much of this period did each seat exist?"* —
summed into a **fractional** seat quantity, so the customer pays exactly for the
time each seat was held.

## Symmetric proration

B is the **symmetric** policy: it prorates in **both** directions.

- **Adding** a seat mid-period → charged only from the day it was added.
- **Removing** a seat mid-period → accrual **stops** the day it leaves; the
  unused remainder is **credited**.

This symmetry is what makes it "fair," and also what makes it the rarer model in
practice — most vendors prorate additions but refuse to credit removals (that
asymmetry is [use case C](./use-case-c-hybrid.md)).

## Real-world examples

**Slack — "Fair Billing Policy"** (the gold standard — prorates adds *and*
credits inactive users)

> "If you're on a paid plan and add new members partway through the billing
> period, we'll only charge for the time used." … "Slack will automatically
> detect if members become inactive, and if that happens, we'll add prorated
> credits to your Slack account."
> — *slack.com/help/articles/218915077*

**GitHub** (prorated on add)

> "Adding seats is prorated and grants immediate access… She's immediately
> charged a prorated amount for June 4–14, and billed for 35 seats starting June
> 15th."
> — *docs.github.com/en/billing/concepts/impact-of-plan-changes*

> Note: GitHub prorates on add but does **not** credit on remove — so GitHub as a
> whole is use case C. It appears here only as a clean example of the
> *prorate-on-add* half.

**Stripe Billing (engine reference)**

> Its whole proration model exists for exactly this: "the customer is charged a
> percentage of a subscription's cost to reflect partial use."

## Worked example

Customer on a **$10/seat/month** plan, 30-day June, billed time-weighted.

| Seat | Days held | Fraction | Cost |
| --- | --- | --- | --- |
| alice | 30 / 30 | 1.00 | $10.00 |
| carol | 30 / 30 | 1.00 | $10.00 |
| bob | 20 / 30 (left on the 21st) | 0.67 | $6.67 |
| dave | 15 / 30 (joined on the 16th) | 0.50 | $5.00 |
| **Total** | | **3.17 seats** | **$31.67** |

Compare: full-period (A) bills 3–4 whole seats; hybrid (C) bills 3.50. B is the
lowest of the three because it is the only one that credits bob's early
departure.

## Maps to

A **carry-over (stock) meter** read with **time-weighting**, with **both**
proration switches on (`prorate_on_increase = true`, `credit_on_decrease =
true`). Each seat's billable interval is clipped to `[join, leave]`. See
[mapping → Use case B](./mapping.md#use-case-b).
