# Seat-based billing

## The problem

You sell software priced **per user** — a flat amount per seat, billed once a
billing period. The reality is that a customer's seat count is not stable:
people are added and removed throughout the period. So when the invoice is cut,
the question *"how many seats do we charge for?"* has no single obvious answer,
because the number was different on different days.

Take one customer's June:

```
        June
        1 ───────────────── 16 ───── 21 ──────────── 30
alice   ●━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━●   present all month
bob     ●━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━✕                left on the 21st
carol   ●━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━●   present all month
dave                          ●━━━━━━━━━━━━━━━━━━━━━━━●   joined on the 16th
```

## The three answers

The same June timeline produces a different charge under each policy:

| Use case | Policy in one line | Charge for the timeline | Character |
| --- | --- | --- | --- |
| **[A — Full-period](./use-case-a-full-period.md)** | A seat present at the moment that matters → pay the **whole** period. | **3 or 4** whole seats (depends on the moment) | Whole numbers. Predictable. Vendor-favourable. |
| **[B — Time-weighted](./use-case-b-time-weighted.md)** | A seat present part of the period → pay **that fraction**. | **3.17** seats | Fractions. Fair. Usage-true. |
| **[C — Hybrid](./use-case-c-hybrid.md)** | **Prorate** when seats are added, **commit** (no credit) when removed. | **3.50** seats | Asymmetric. The real-world default. |

The contrast in one line:

- **Full-period (A):** a seat present at all → pay the whole period.
- **Time-weighted (B):** a seat present part of the period → pay that fraction.
- **Hybrid (C):** prorate on the way *up*, commit on the way *down*.

These are not rival companies — they are frequently the **same vendor's monthly
vs annual plans**. The dominant real-world pattern is the asymmetric hybrid (C);
pure symmetric proration with credits (B, e.g. Slack) is the generous outlier.
Each use-case doc cites the vendors that ship it.
