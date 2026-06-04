# Invoice-centric billing; subscription carries no charge amount

**Context.** Today a renewal charges `subscription.Amount` — a single flat `int64` —
directly through the gateway, recording only a `Payment`. There is no `Invoice` and no
per-cycle itemization. This can't express usage billing, where the period amount is
variable and unknowable up front (a usage/seat subscription has no meaningful fixed
amount).

**Decision.**

1. Introduce **`Invoice`** as a new entity, generated on **every** billing run for
   **all** subscriptions (fixed and metered). It holds line items — a base line from
   the linked `Price`, plus usage lines from meters — and a calculated total.
2. `billing-cycle` changes from "charge `subscription.Amount`" to **"build the invoice
   → total it → create a `Payment` that settles that total."** The `Payment` remains
   the record of a PSP attempt; the `Invoice` is the record of what was owed.
3. **`Subscription` no longer stores a charge amount.** Pricing authority is the linked
   `Price`(s) + `Meter`; the subscription keeps only the agreement and historic actuals
   (revenue to date, cycles processed). A "base MRR" figure, if ever needed, is derived
   on demand (the fixed-price slice), not stored.

**Why.** Usage billing is meaningless without an itemized, variable, per-period record,
and a single stored amount is a fiction for metered subscriptions. Both reference
systems we studied confirm the shape: Lago keeps no amount on the subscription (price
lives on the plan + charges; periods are billed into fees/invoices); Polar's
subscription amount is only the derived fixed slice, with metered prices contributing
zero and usage billed separately.

**Consequences.**

- A real refactor: every path that reads `subscription.Amount` as the charge must move
  to computing the amount from the invoice. Backward compatibility was waived.
- Supersedes the original ADR 0001 framing (the decimal-quantity line item lives on the
  **invoice** line item, not `OrderItem`).
- This is now broader than "add metering" — usage metering depends on this billing
  model, so the Invoice work is its own piece, not part of the metering spec alone.
