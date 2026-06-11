# Package pricing — worked use case (SMS resale)

A reference walkthrough of how the billing model handles **package** pricing,
traced through the actual domain code (`internal/core/domain/pricing.go`,
`price.go`, `price_types.go`).

## Use case

A merchant resells SMS on your platform. They buy capacity wholesale in bundles
of 1,000 messages and sell at **$5 per started 1,000 SMS** — billed per block,
not per message. A customer who sends 12,400 SMS in June owes:

```
blocks = ceil(12,400 / 1,000) = 13
total  = 13 × $5.00           = $65.00
```

Why blocks instead of a linear $0.005/message:

- **Cost symmetry** — the merchant's own cost is incurred per wholesale bundle,
  so revenue rounds the same direction as cost. Linearly, a customer sending
  200 messages pays $1 while the merchant burned a full bundle.
- **No absurd invoices** — a trial customer who sends 37 messages produces a
  $5 line item, not an 18¢ invoice that costs more in gateway fees than it
  collects.
- **Pricing-page readability** — "$5 per 1,000 SMS" beats "$0.005 per SMS".

The tell that a merchant needs **package** rather than **fixed**: they'd be
upset if an invoice came to $3.70. If a partial block should prorate, it's
fixed; if a started block owes the full block, it's package.

## How it's modeled

- **`Price.Scheme = Package`** — `price_types.go`
- **`Price.UnitPrice`** = cents per block (`500`), **`Price.UnitCount`** = block
  size (`1000`) — the *same pair* the fixed scheme uses for "$1 per 1000 calls".
  `UnitPrice` always means "cents per `UnitCount` units" in both schemes; the
  only difference is what happens to a partial block. (Lago's `amount` key
  silently switches denominator between its standard and package charge models —
  the explicit `UnitCount` denominator avoids that trap.)
- Computed by **`pricePackage`** (`pricing.go`), dispatched from `PriceUsage`:
  `ceil(units / UnitCount) × UnitPrice`, rounded once via `roundWithUnit`.
- **Metered only** — package bills started blocks of *usage*, so it requires a
  `BillableMetricId` (`validatePriceConfig`, `internal/core/service/product.go`).
  Cart/order/base-line paths never see it.
- **Flat by definition** — block #1 costs the same as block #500; tiers are
  rejected. If the rate should change with volume, that's graduated or volume.

### Config — the `Price`

```go
Price{
    Label:            "SMS",
    Scheme:           Package,               // started block owes the full block
    Currency:         "USD",
    UnitPrice:        500,                   // $5 per block, in cents
    UnitCount:        1000,                  // block size, in units
    BillingInterval:  BillingIntervalMonth,  // metered ⇒ capped at monthly anyway
    BillableMetricId: "<sms-sent meter>",    // required for package
}
```

## Diagrams

### 1. The staircase (price jumps at every started block)

```
total $
  15 ┤                          ┌────────
     │                          │
  10 ┤              ┌───────────┘
     │              │
   5 ┤  ┌───────────┘                       package: every started
     │  │ ← 37 SMS already owes $5          block owes the full $5
   0 └──┴───────────┬───────────┬─────────▶ SMS sent
     0            1,000       2,000

     fixed (same UnitPrice/UnitCount) would draw a straight line
     through the step corners — prorating the partial block.
```

### 2. June's 12,400 SMS

```
blocks  │████│████│████│████│████│████│████│████│████│████│████│████│▌   │
        1    2    3    4    5    6    7    8    9    10   11   12   13
        ◀──────────────── 12 full blocks ───────────────────▶ ◀ 400 SMS
                                                                started
        13 blocks × $5 = $65.00                                block 13 ⇒
                                                                full $5
```

### 3. How `pricePackage` computes it (`pricing.go`)

```
units = 12,400
  │
  ├─ units ≤ 0 ?  no → continue            (zero usage owes nothing — no minimum block)
  ├─ size = max(1, UnitCount) = 1,000
  ├─ blocks = ceil(12,400 / 1,000) = 13
  ▼  roundWithUnit(13 × 500¢, 12,400)
     amountCents     = 6,500              → $65.00
     unitAmountCents = 6,500 / 12,400 ≈ 0.524¢   (blended effective rate)
```

The invoice line carries the rolled-up total plus the effective per-unit rate —
same convention as graduated.

### 4. Package vs fixed (same config, same 12,400 — the partial block decides)

```
PACKAGE (this use case)              FIXED (prorating sibling)
ceil(12,400/1,000) = 13 blocks       12,400 × 500/1,000
13 × $5  = $65.00                    = $62.00
▲ started block owes the block       ▲ partial block prorates exactly
  (revenue rounds like wholesale       (right for $1-per-1000-API-calls
  cost; no 18¢ invoices)               where there is no block cost)
```

Same `(UnitPrice, UnitCount)` pair; only the rounding of the last block
differs. Pinned by `TestPriceUsage_PackageVsFixedPartialBlock`
(`pricing_package_test.go`).

### 5. Where package sits among the schemes

```
                    flat rate                rate varies with volume
                ┌────────────────────┬──────────────────────────────┐
prorate partial │ fixed              │ graduated (band by band)     │
                │   q × price/count  │ volume    (one band for all) │
                ├────────────────────┼──────────────────────────────┤
round partial   │ package            │ — (not composed; tiers and   │
block up        │   ceil(q/count)×p  │    block-rounding stay       │
                │                    │    separate, as in Lago/     │
                │                    │    Stripe)                   │
                └────────────────────┴──────────────────────────────┘
```

"First 100 free, then per-unit" already has a home: graduated with a zero-rate
first tier. Package deliberately does **not** take a free-units knob or tiers —
that keeps `UnitPrice` meaning exactly one thing.

## Same pattern elsewhere

Anywhere usage is consumed in small units but provisioned in chunks: email
sends sold in blocks, verification/KYC lookups ("$50 per 100 checks"), AI
tokens sold as credit packs, storage billed per started GB.

## Related

- Fixed-scheme `unit_count` (the prorating sibling): `Price.UnitCount`,
  `priceFixed` (`pricing.go`).
- Graduated worked example: `graduated-use-case.md`.
- The quantity→rate pipeline: `billing-model/stock-billing-architecture-impact.md` §1.
- Metered cadence is capped at monthly (`Price.SubscriptionCadence`, `price.go`).
