# Order → Subscription HTTP driver

**Date:** 2026-06-03
**Goal:** Give a developer a runnable way to set up an order and start a subscription end-to-end against the local stack, to verify the flow works.

## Problem

The codebase has a full order→subscription flow (create org → customer → product with a
`subscription` price → order → complete order, which activates the subscription and starts the
billing workflow), exercised only by a unit test (`order_test.go`). There is no runnable
end-to-end driver and no easy way to authenticate ad-hoc API calls.

## Decision

Deliver a REST Client `.http` file that drives the flow, authenticated with an API key.

### Auth: API key

Requests carry the key in `x-api-key`. The key is org-scoped, so **org creation is skipped** —
every call is auto-scoped to the key's org.

One required code change: the `apikey.ApiKeyMiddleware` exists and works but is not wired. Add it
to the authenticator slice in `internal/config/app.go` (after Clerk):

```go
apiKeyAuth := apikey.NewApiKeyMiddleware(logger, env, apiKeyRepo)
authenticators := []port.Authenticator{clerkAuth, apiKeyAuth}
```

`AuthnWrapperMiddleware` tries authenticators in order and the first success wins; Clerk fails
cleanly on a non-Clerk token and falls through to the key. Requires `API_KEY_PEPPER` set to the
same value used when the key was minted.

### No live PSP charge required

`OrderService.CreateOrder` only calls the payment gateway when a `session_id` is supplied
(`order.go:81-85`, `createPspSession`). Passing a **direct cart** skips the gateway entirely.
`psp_id` is `required` by request validation but unused on that path, so a dummy value passes.
`CompleteOrder` does no charge — it records the first payment, activates the subscription, and
fires `StartSubscriptionWorkflow` as a **post-commit** side effect. Payment-method creation
(`CustomerService.CreatePaymentMethod`) stores the token without contacting the gateway, but
requires a billing address on the customer or the payment-method input.

## Flow (`requests/order-subscription.http`)

1. `POST /api/customers` — email, name, billing address → capture `customerId`
2. `POST /api/products` — one variant, one price `category: subscription`, `billing_interval: month`
   → capture `productId`, `priceId`
3. `POST /api/customers/{customerId}/payment-methods` — `type: card`, dummy token → capture `paymentMethodId`
4. `POST /api/orders` — direct `cart` (currency + product/price/qty), existing customer, dummy
   `psp_id`, the `payment_method_id` → capture `orderId`
5. `POST /api/orders/{orderId}/complete` — first payment amount > 0 → activates subscription
6. `GET /api/orders/{orderId}/subscriptions` — **verify** subscription status is `active`

## Scope

Wire the authenticator, build the `.http` file, then run it live against the local stack until
step 6 shows `active`. With Hatchet up the recurring-billing workflow also starts; if it is down,
the subscription still commits as `active` (workflow start is post-commit) — that gap is noted but
not blocking for the data-level verification.

## Outcome (2026-06-03)

Flow verified end-to-end against the local stack (API on `:10081`, not `:8080` — `SERVER_PORT=10081`):
order completes (`status: completed`), subscription lands `active`, and the durable
`subscription-runner` workflow starts in Hatchet. Driving the flow surfaced five real,
pre-existing defects that blocked it — all fixed:

1. **API-key auth not wired** — `app.go` only had Clerk; added the `apikey` authenticator.
2. **Hatchet-lite stale DB connection** (environment) — its pooled conn to Postgres had gone
   dead after a long idle, so worker registration failed with `context canceled`; fixed by
   restarting the `hatchet-lite` container. Also cleared a leftover server process holding
   `:10081`.
3. **`CustomerService.Create` treated "no existing customer" as a 500** — `FindByEmail`
   returns a not-found error on the happy path; now ignored via `errors.Is(err, port.ErrNotFound)`.
4. **Nullable FK written as empty string** — `customers.default_payment_method_id` (a nullable
   FK) got `''` instead of `NULL`; `CustomerRepo` now omits the column when empty.
5. **Nil `map[string]string` → SQL NULL on NOT-NULL `metadata` columns** — orders, order_items,
   and subscriptions; postgres repos now default nil metadata to `{}` via `emptyIfNil`.
6. **`OrderItem.VariantId` never populated** — violated `order_items_org_id_variant_id_fkey`;
   threaded the variant id through `CartItemPrice` → `OrderItem`.

Non-blocking: a best-effort "add customer to cohort" step logs an FK error
(`customer_cohorts_org_id_cohort_id_fkey`) when no cohort is configured. Out of scope for this
flow; left as-is.

The driver lives at `requests/order-subscription.http`.
