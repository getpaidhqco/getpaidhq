# Invoice line items use a decimal quantity

Per-cycle billing is itemized on an **Invoice** (ADR 0002), and usage lines carry
fractional quantities (e.g. 41.6667 seat-hours, 1.2 GB). So an invoice line item stores
`Quantity` as a **decimal** (product lines are whole numbers, usage lines fractional —
one field, as Lago does with fee `units`), plus a sub-cent-capable `UnitAmount`
(decimal cents) for the per-unit rate, since usage rates can be below a cent ($0.001).
`Total` stays `int64` cents — the actually-charged amount, rounded once.

This decision originally targeted `OrderItem.Quantity`; it was retargeted to the
invoice line item once we established that per-cycle charges live on the Invoice, not on
a per-cycle order (ADR 0002). `OrderItem` is left unchanged.
