# Use case C — Hybrid seat billing (prorate up, commit down)

> **[← Index](./README.md)** · Siblings: [A — Full-period](./use-case-a-full-period.md) · [B — Time-weighted](./use-case-b-time-weighted.md) · [How we bill it →](./mapping.md#use-case-c)

## Definition

A seat added mid-period is **prorated** — charged only from the day it was added.
A seat removed mid-period is **committed** — it keeps accruing to the end of the
period, with **no credit** for the unused remainder.

This is the **asymmetric** policy: generous on the way up, firm on the way down.

The business is saying: **"Add people whenever you like and only pay for the time
they're here — but once you've taken a seat for the period, it's yours for the
period."** It captures most of the fairness of [time-weighted billing](./use-case-b-time-weighted.md)
on additions (the common, friction-sensitive action) while protecting revenue
against mid-period churn.

```
alice   ████████████████████████  full month               →  1.00
carol   ████████████████████████  full month               →  1.00
bob     ████████████████████████  left on the 21st, COMMITTED →  1.00   (no credit)
dave    ░░░░░░░░░░░░████████████  joined 16th, PRORATED     →  0.50
                                                              ─────
                                                charged for   3.50 seats
```

The question it answers: *"From when did each seat start accruing, and does
leaving stop it?"* — additions start the clock at the join date; removals never
stop it before period end.

## Why this is the real-world default

The two pure policies (A and B) are textbook endpoints. The pattern most large
per-seat vendors actually ship is this **asymmetric hybrid** — often *within a
single plan*:

| Vendor | On add | On remove |
| --- | --- | --- |
| **GitHub** | prorated (immediate access, charged pro-rata) | takes effect next cycle, no mid-cycle refund |
| **Atlassian** (monthly) | prorates additions | "there won't be any deductions, refunds or credits" |
| **Microsoft 365** (NCE) | prorated | annual reductions blocked entirely |
| **Google Workspace** (Flexible) | "you pay only for the accounts you have during a month" | reductions stop billing, but the Annual plan is pure commit |

So the dominant shape is: **prorate on the way up, commit/full-period on the way
down.** Pure symmetric proration with credits ([Slack](./use-case-b-time-weighted.md#real-world-examples))
is the generous outlier, not the norm.

> The same vendor often offers A and C side by side as **annual vs monthly**
> plans (Google, Microsoft). A is "annual commitment"; C is "monthly flexible."

## Worked example

Customer on a **$10/seat/month** plan, 30-day June, billed hybrid.

| Seat | Treatment | Fraction | Cost |
| --- | --- | --- | --- |
| alice | full month | 1.00 | $10.00 |
| carol | full month | 1.00 | $10.00 |
| bob | left on the 21st — **committed**, no credit | 1.00 | $10.00 |
| dave | joined on the 16th — **prorated** | 0.50 | $5.00 |
| **Total** | | **3.50 seats** | **$35.00** |

The three policies on one timeline:

```
B  time-weighted   3.17 seats   $31.67   ← fairest, credits bob's departure
C  hybrid          3.50 seats   $35.00   ← prorates dave's join, commits bob
A  full-period     3 or 4 seats $30–40   ← whole seats, ignores partial time
```

C sits between B and A: it grants dave the proration of B but denies bob the
credit, landing above B and below the distinct-ever-active reading of A.

## Maps to

A **carry-over (stock) meter** read with **time-weighting**, with the proration
switches set **asymmetrically**: `prorate_on_increase = true`,
`credit_on_decrease = false`. A seat's billable interval starts at its join date
but always extends to period end. See [mapping → Use case C](./mapping.md#use-case-c).
