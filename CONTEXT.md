# GetPaidHQ (gphq-server)

Subscription-billing backend. This glossary covers the domain language; usage-based
metering terms are being added as the design is grilled.

## Language

### Metering

**Billable Metric** (aka **Meter**):
A definition of *what* customer usage to measure and *how* to add it up over a period
(count, sum, max, etc.).
_Avoid_: "metric" alone (overloaded with monitoring/metrics).

**Usage Event**:
A single recorded use, belonging to a **Customer** and a **Billable Metric**. It is
not tied to a **Subscription** at record time.
_Avoid_: "meter event" in prose (that's the table name), "usage record".

**External Customer ID**:
The merchant's own identifier for a **Customer**, supplied on usage events. It matches
a Customer's `external_id`. Immutable once set, so it's a stable join key.
_Avoid_: confusing with the event's own dedup id (see **Event External ID**).

**Event External ID**:
The caller's own id for a single **Usage Event**, used as the dedup key. Distinct from
**External Customer ID**.
_Avoid_: "transaction id", "idempotency key" (those imply a payment/transaction).

**Metered Price**:
A **Price** of category `metered` that links a **Billable Metric** to a **Pricing
Scheme**.

**Pricing Scheme**:
How a quantity of units becomes money on a **Price**. Three models:
- **Fixed** — flat rate: `amount = units × unit price`.
- **Graduated** — progressive tiers: each unit billed at the rate of the tier it falls
  into; total summed across tiers.
- **Volume** — all units billed at the single tier the *total* quantity reaches.
_Avoid_: **Tiered** (collapsed into Graduated — same thing), **Package** (not modelled).

**Unattributed usage**:
Usage events with no subscription named. Billed by the customer's *earliest* active
metered **Subscription** for that meter.

### Billing records

These four are the spine; they were being conflated and are pinned here.

**Order**:
The record that a customer made a purchase — what was bought (its **Order Items**:
products at prices) and for whom. Carries the *agreed pricing snapshot*; for usage
pricing that snapshot is only an estimate, not a charge. The genesis record; a
**Subscription** is created from an Order Item. One-time.
_Avoid_: using "Order" for anything recurring or per-cycle (that's the **Invoice**).

**Subscription**:
A recurring agreement created from an **Order Item**, linked to a product/**Price**
(and, for metered, a **Meter**). It does **not** store the charge amount — for usage
pricing the amount isn't knowable up front, so each cycle's amount is computed into an
**Invoice** from the linked **Price**(s) + usage. The Subscription holds the agreement
plus *historic actuals* (revenue to date, cycles processed).
_Avoid_: a `subscription.amount` field used as the charge source.

**Invoice** (new — does not exist yet):
The per-billing-run record of what is owed: the calculated line-item totals for one
period (base line from the linked Price + any usage lines). Generated on **every**
cycle. Its total is the amount a **Payment** attempts to settle.
_Avoid_: "bill", "receipt".

**Invoice preview** (aka **pro forma**):
A *computed* estimate of an upcoming or in-progress **Invoice** — e.g. usage-so-far.
Derived on demand, never stored.
_Avoid_: "quote".

**Payment**:
A record of one charge *attempt* at the PSP (status, fees, net amount). Settles an
**Invoice**.
_Avoid_: treating a Payment as the calculated total (that's the Invoice).

### Existing domain (referenced)

**Customer**:
A person or organization that is billed. Gains an `external_id` (the **External
Customer ID**) for usage attribution.
_Avoid_: "account", "user" (User is a separate concept — an operator of an Org).
