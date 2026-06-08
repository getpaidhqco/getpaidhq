# Product archiving instead of soft-delete

A `Product` cannot always be deleted: the `order_items → variants` foreign key is
intentionally `Restrict` so order history is preserved, which means a hard
`DELETE /products/{id}` returns **409** for any product that has ever been sold. That left
merchants with no way to retire a product — it stayed visible in the dashboard forever.

We deliberately **do not** add soft-delete (a `deletedAt` tombstone that filters rows out
everywhere). Instead a product carries an explicit lifecycle **status**:

- **active** — live and **sellable**; shown in default catalog listings (dashboard, checkout).
- **archived** — retired: hidden from default listings and **not sellable**, but fully
  preserved in historic data (orders, invoices, subscriptions, reports). Reversible.

There are exactly two states — no `draft` or other status — and "sellable" is 1:1 with
`active`. A nullable `archivedAt` timestamp records *when* a product was retired (audit /
reporting); `status` remains the queryable source of truth.

## What this means in the API

- `POST /products/{id}/archive` and `POST /products/{id}/unarchive` flip the status. They are
  idempotent (archiving an archived product is a 200 no-op) and reuse the `UpdateProduct`
  permission — archiving is a privileged product mutation, not a separate grant.
- `GET /products` returns **active only by default**. `?status=` selects the view:
  `active` (default) · `archived` · `all`. Any other value is a **400**. The OpenAPI spec
  (Fuego-generated) types the field as a plain string and does not emit the enum constraint,
  so the allowed values are documented here.
- Single-resource reads are unaffected — `GET /products/{id}`, its variants,
  `GET /variants/{id}`, `GET /prices/{id}` return archived products normally (needed to view,
  unarchive, and resolve historic references).
- Archived products are **editable** (PATCH name/metadata, add/edit variants and prices);
  archiving only governs visibility and sellability, not mutability.
- Hard `DELETE` is **kept** for genuinely-orphan products (created by mistake, no orders); it
  still 409s when history exists. Archiving is the path for anything that has been sold.

## Sellability is enforced at checkout only

Adding an archived product to a **cart**, or creating an **order** that references one, is
rejected with **409 Conflict** (`CartService.AddProduct`, `OrderService.CreateOrder`).
Because subscriptions are only created via the order flow, this also blocks new
subscriptions. Crucially, **recurring billing is untouched**: renewals run
`Subscription → Invoice → Payment` from the frozen price/order snapshot and never call
`CreateOrder` or load the product, so archiving a product does not stop billing for customers
who already bought it.

## Events

Archiving emits a distinct `product.archived` / `product.unarchived` event (not a generic
`product.updated`), matching the existing per-entity `created/updated/deleted` event grain so
consumers can react to the lifecycle transition precisely.

## Trade-off

Soft-delete would hide a product everywhere with one flag, but it conflates "gone" with
"retired", needs a global filter on every query, and still can't truly remove rows tied to
history. Archiving is an explicit, reversible, reportable lifecycle state that keeps historic
data intact and confines the behavioural change to listings and the point of sale.
