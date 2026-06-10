# Use case A — Full-period seat billing (not prorated)

> **[← Index](./README.md)** · Siblings: [B — Time-weighted](./use-case-b-time-weighted.md) · [C — Hybrid](./use-case-c-hybrid.md) · [How we bill it →](./mapping.md#use-case-a)

## Definition

A seat is charged the **full** period price if it was present **at the moment
that matters** — regardless of how long it was actually held during the period.
Partial presence is not measured; there is no concept of a fraction of a seat.
The customer pays a **whole number of seats**, each at full price.

The business is saying: **"A seat is a seat. If you used it this period, you pay
for the period."** This is simpler to understand, simpler to predict, and biases
toward the vendor — mid-period joiners pay in full, and (depending on the chosen
policy) a seat may still be billed for the whole period even if it was given up
partway through.

The period is treated as **one unit**. Each seat is "in" or "out" — no fractions.

```
alice   ████████████████████████  full seat   →  1
carol   ████████████████████████  full seat   →  1
bob     ████████████████████████  full seat   →  1   (present during the period)
dave    ████████████████████████  full seat   →  1   (present during the period)
                                                 ───
                                    charged for   4 whole seats
```

The question it answers: *"How many distinct seats did this customer have this
period?"* — counted as whole seats, billed at full price, with no credit or
charge for partial time.

## which moment counts?

"Present at the moment that matters" is deliberately vague, because there are
three defensible moments — and they give **different numbers** on the same
timeline. Using the [June timeline](./README.md#the-problem):

| Moment | Definition | June result | Who counts |
| --- | --- | --- | --- |
| **End-of-period** | Seats active at period close. | **3** | alice, carol, dave (bob left on the 21st) |
| **Peak concurrent** | The most seats held *simultaneously* at any instant. | **4** | all four overlap between the 16th and the 21st |
| **Distinct ever-active** | Every distinct seat that existed at *any* point. | **4** | alice, bob, carol, dave |

- **End-of-period** is the most intuitive "you pay for the seats you have." A
  seat added then removed within the period is never billed.
- **Peak concurrent** is vendor-favourable and punishes spikes — a short burst
  of seats sets the bill for the whole period.
- **Distinct ever-active** bills anyone who showed up at all; it never forgets a
  seat once seen. (This is the "4 seats" answer in the diagram above.)

Each moment is one of the meter's **aggregations**, read over the standing seat
level: end-of-period = `latest`, peak concurrent = `max`, distinct ever-active =
`unique_count`. See the [mapping](./mapping.md#axis-1-whole-seat-counting) for how
each is computed.

## Real-world examples

**Google Workspace — Annual / Fixed-Term Plan**

> "You commit to purchasing the subscription for one or multiple years… You can
> reduce licenses only when renewing your plan at the end of the contract. Until
> then, you pay for all purchased licenses."
> — *knowledge.workspace.google.com/admin/billing/compare-flexible-and-annual-fixed-term-payment-plans*

**Microsoft 365 — New Commerce Experience (annual term)**

> "You can only remove licenses from your subscription if it's within seven days
> of buying or renewing… If you reduce the number of licenses after that seven
> day period, the change appears on the first invoice you receive after the
> subscription renewal date."
> — *learn.microsoft.com/en-us/microsoft-365/commerce/licenses/buy-licenses*

**Zoom (backup)**

> "A reduction in license quantities will take effect at the end of your current
> billing cycle, not immediately… You will continue to have access to all
> licenses until your renewal date."

These are **committed-seat** models: the count is fixed for the term, reductions
do not take effect mid-period, and there is no credit for a seat given up early.

## Worked example

Customer on a **$10/seat/month** plan, billed on the June timeline.

- Policy: full-period, **end-of-period** moment.
- Seats active at June 30: alice, carol, dave → **3 seats**.
- Invoice line: `3 × $10 = $30.00`, quantity `3` (a whole integer).

Switch the moment to **distinct ever-active** and the same timeline bills
`4 × $10 = $40.00`. Same events, different policy, different revenue.

## Maps to

A **carry-over (stock) meter** with a **whole-seat aggregation**
(`latest` | `max` | `unique_count`) and **no proration**.
See [mapping → Use case A](./mapping.md#use-case-a).
