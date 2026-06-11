# Graduated pricing — worked use case (transactional email API)

A reference walkthrough of how the billing model handles **graduated** pricing,
traced through the actual domain code (`internal/core/domain/pricing.go`,
`price.go`, `price_types.go`).

## Use case

A transactional email API. You want big senders to **still pay full price on their
first emails**, with the rate easing as volume climbs — so early usage protects
revenue and heavy usage earns a marginal discount.

The tiers (graduated = each slice billed at its own rate):

| Tier | Range (emails)     | Rate each |
| ---- | ------------------ | --------- |
| 1    | 0 – 10,000         | $0.0010   |
| 2    | 10,001 – 100,000   | $0.0005   |
| 3    | 100,001 +          | $0.0002   |

This period the customer sent **250,000 emails**. You fill each bucket in order and
charge that bucket's rate:

```
Tier 1:   10,000 emails  × $0.0010 = $10.00     (the first 10k)
Tier 2:   90,000 emails  × $0.0005 = $45.00     (10,001 → 100,000)
Tier 3:  150,000 emails  × $0.0002 = $30.00     (100,001 → 250,000)
                                      ──────
                          total      $85.00
```

## How it's modeled

- **`Price.Scheme = Graduated`** (`"tiered"` is an alias) — `price_types.go:33`
- **`Price.Tiers []PriceTier`** holds the rate bands — `price.go:26`
- Each **`PriceTier`** (`pricing.go:7`): `FromValue` (inclusive lower bound),
  `ToValue` (upper bound; `0` = unbounded last tier), `PerUnitAmount` (cents/unit,
  sub-cent allowed via `decimal.Decimal`), `FlatAmount` (flat cents added when the
  tier is used).
- Computed by **`priceGraduated`** (`pricing.go:32`), dispatched from `PriceUsage`.

Amounts are **cents**, and `PerUnitAmount` is `decimal.Decimal`, so sub-cent rates
are exact: $0.0010 = `0.1`¢, $0.0005 = `0.05`¢, $0.0002 = `0.02`¢.

### Config — the `Price`

```go
Price{
    Label:            "Transactional Email",
    Scheme:           Graduated,             // each slice billed at its own rate
    Currency:         "USD",
    BillingInterval:  BillingIntervalMonth,  // metered ⇒ capped at monthly anyway
    BillableMetricId: "<emails-sent meter>", // IsMetered() ⇒ true
    Tiers: []PriceTier{
        {FromValue: dec(0),       ToValue: dec(10_000),  PerUnitAmount: dec(0.1)},  // $0.0010
        {FromValue: dec(10_000),  ToValue: dec(100_000), PerUnitAmount: dec(0.05)}, // $0.0005
        {FromValue: dec(100_000), ToValue: dec(0),       PerUnitAmount: dec(0.02)}, // $0.0002, 0=unbounded
    },
}
```

Key detail: `FromValue` is the **inclusive lower bound** and
`qty = min(units, ToValue) − FromValue`. Tiers **abut** (10,000, 100,000) rather
than offsetting by 1 — the boundary unit isn't double-counted because tier *N*'s
`ToValue` equals tier *N+1*'s `FromValue`. No `FlatAmount` in this use case.

## Diagrams

### 1. The tier ladder (rate eases as volume climbs)

```
rate ¢/email
0.10 ┤████ tier 1
     │   █ 0 – 10k                    @ $0.0010
0.05 ┤   └──────────┐
     │              █ tier 2
     │              █ 10k – 100k      @ $0.0005
0.02 ┤              └────────────────┐
     │                               █ tier 3
     │                               █ 100k +          @ $0.0002
  0  └───┬──────────┬────────────────┬──────────────▶ cumulative emails
        10k       100k             250k
   big senders pay FULL rate here ──┘  heavy usage gets the discount ──┘
```

### 2. The 250,000 emails sliced across tiers (each slice at its own price)

```
           units in band         rate        amount
tier 1  │█│               10,000   × $0.0010 = $10.00
tier 2  │█████████│       90,000   × $0.0005 = $45.00
tier 3  │███████████████│ 150,000  × $0.0002 = $30.00
        └─────────────────────────────────────────────
        0       100k             250k          $85.00
        ◀── full price ──▶◀──── marginal discount ────▶
```

### 3. How `priceGraduated` fills the buckets (the loop in `pricing.go:32`)

```
units = 250,000
  │
  ▼  for each tier:  qty = min(units, ToValue) − FromValue   (skip if ≤ 0)
  │
  ├─ tier{From 0,      To 10k }  hi=min(250k,10k)=10k    qty= 10,000  ┐
  │      total += 10,000 × 0.1¢  = 1,000¢                             │ units > To
  │                                                                    │ → continue
  ├─ tier{From 10k,    To 100k}  hi=min(250k,100k)=100k  qty= 90,000  ┐
  │      total += 90,000 × 0.05¢ = 4,500¢   (running 5,500¢)          │ units > To
  │                                                                    │ → continue
  ├─ tier{From 100k,   To 0    }  ToValue=0 ⇒ unbounded                ┐
  │      hi=units=250k            qty=150,000                          │ unbounded
  │      total += 150,000 × 0.02¢= 3,000¢   (running 8,500¢)          │ → break
  │                                                                    ┘
  ▼  roundWithUnit(8,500¢, 250,000)
     amountCents     = 8,500            → $85.00
     unitAmountCents = 8,500 / 250,000  = 0.034¢   (blended effective rate)
```

Total is rounded **once** at the end (`roundWithUnit`), so sub-cent tier rates
never lose precision mid-sum.

### 4. Data model → invoice

```
        Variant
           │ 1
           ▼ *
        Price                                  PriceUsage(price, units)
   ┌───────────────────────────┐                        │
   │ Scheme  = Graduated        │                       ▼
   │ Metric  = emails-sent ─────┼──────▶ usage = 250,000
   │ Interval= month (capped)   │                        │
   │ Tiers[] ───────────┐       │                        ▼
   └────────────────────┼───────┘                 ┌──────────────┐
        ┌───────────────┼───────────┐             │ amountCents  │ = 8,500
        ▼               ▼           ▼             │ unitAmount¢  │ = 0.034
   PriceTier       PriceTier   PriceTier          └──────┬───────┘
   From 0          From 10k    From 100k                 │
   To  10k         To  100k    To  0 (∞)                 ▼
   0.1¢            0.05¢       0.02¢          ┌───────────────────────────────┐
                                              │ Invoice line                  │
                                              │ Transactional Email   $85.00  │
                                              │   one rolled-up metered line  │
                                              └───────────────────────────────┘
```

The line carries the rolled-up `$85.00` (`amountCents`) plus the effective unit
rate; the per-tier breakdown is the derivation, not three separate stored lines.

### 5. Graduated vs Volume (same tiers, same 250k — why this use case picks graduated)

```
GRADUATED (this use case)            VOLUME (whole qty at one tier's rate)
each slice its own rate              250k lands in tier 3 ⇒ ALL at 0.02¢
 10k × 0.10¢ = $10                   250,000 × $0.0002 = $50.00
 90k × 0.05¢ = $45
150k × 0.02¢ = $30
 ──────────── $85.00                 ─────────────────── $50.00
 ▲ early usage protects revenue      ▲ big sender gets discount on ALL units
                                       (first emails NOT full price — wrong here)
```

**Graduated is the correct scheme** for "big senders still pay full price on their
first emails" — volume would retroactively discount the whole quantity
(`priceVolume`, `pricing.go:53`).

## Related

- Filters vs groups (splitting a priced line for visibility, e.g. per `project` or
  `api_key`, at the **same** rate): `billing-model.md`.
- Package worked example (flat block pricing — a started block owes the full
  block): `package-use-case.md`.
- Metered cadence is capped at monthly (`Price.SubscriptionCadence`, `price.go:59`).
