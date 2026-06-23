# Hatchet Complete-Order Hardening

## Goal

Make the Hatchet `payment-success` complete-order step safe to retry.

The desired guarantee is:

> One PSP successful payment completes one order at most once, even if the webhook is retried, the Hatchet task retries, or the workflow is started twice.

## Scope

In scope:

- Add deterministic Hatchet run keys for payment workflows.
- Wrap webhook-driven order completion in a DB transaction.
- Lock the order row while completing it.
- Publish `order.completed` only after the DB transaction commits.


## Current Problem

The Hatchet `payment-success` workflow is already a DAG:

1. `complete-order`
2. `get-subscriptions`
3. `start-subscription-lifecycle`

The weak point is inside `complete-order`.

`complete-order` calls `OrderWorkflowService.CompleteCheckoutSession`, which currently performs several writes without a transaction:

- mark order completed
- create payment method
- update subscriptions
- create payment
- publish `order.completed`

If the task fails halfway through, Hatchet retries the whole task. Without transaction and idempotency boundaries, a retry can repeat inner side effects.

## Design

### 1. Key Hatchet payment workflow runs

Add run keys in the Hatchet adapter when starting payment workflows.

For payment success:

```text
payment_success:{orgId}:{orderId}:{psp}:{paymentIdentity}
```

For payment refunded:

```text
payment_refunded:{orgId}:{orderId}:{psp}:{paymentIdentity}
```

`paymentIdentity` should be:

1. `Payment.PspId`, if present.
2. `Payment.Reference`, if `PspId` is empty.
3. Error if both are empty.

Add helpers in `internal/adapter/hatchet/workflows/keys.go`:

```go
func PaymentSuccessRunKey(orgId, orderId string, psp domain.Gateway, paymentIdentity string) string
func PaymentRefundedRunKey(orgId, orderId string, psp domain.Gateway, paymentIdentity string) string
```

Use them in `internal/adapter/hatchet/hatchet.go` with `hatchet.WithRunKey(...)`.

### 2. Add an order row lock

Extend `port.OrderRepository`:

```go
FindByIdForUpdate(ctx context.Context, orgId string, id string) (domain.Order, error)
```

Implement it for both storage adapters:

- GORM: `SELECT ... FOR UPDATE` via `clause.Locking{Strength: "UPDATE"}`.
- pgx: append `FOR UPDATE` to the order lookup query.

This method is only for use inside `TxManager.RunInTx`.

### 3. Make `CompleteCheckoutSession` atomic

Inject `port.TxManager` into `OrderWorkflowService`.

Inside `CompleteCheckoutSession`, wrap all DB mutations in `RunInTx`.

Inside the transaction:

1. Lock the order with `FindByIdForUpdate`.
2. If order is already `completed`, return success as an idempotent no-op.
3. If order is not `pending`, return a permanent conflict error.
4. Create the payment method.
5. Mark the order completed.
6. Load subscriptions.
7. Update subscriptions.
8. Create the payment row when amount is greater than zero.

All repository calls inside the transaction must use the transaction callback context.

### 4. Publish after commit

Move `pubsub.Publish(order.completed)` outside the transaction.

Rules:

- Publish only after `RunInTx` returns nil.
- Do not publish when the transaction fails.
- Do not publish for the already-completed idempotent no-op.
- Log publish failure, but do not return it as the `complete-order` task error.

This prevents a publish failure from causing Hatchet to retry the DB mutation.

## Error Semantics

Permanent errors should not consume Hatchet retries.

Permanent examples:

- order not found
- order not pending and not already completed
- missing PSP payment identity
- invalid payment context

Retryable examples:

- DB connection failure
- deadlock or serialization failure
- transient repository failure

The core service should return normal domain errors. The Hatchet workflow task should translate permanent errors to Hatchet non-retryable errors.

## Acceptance Criteria

- Starting the same payment-success workflow twice uses the same Hatchet run key.
- Starting different PSP payments uses different run keys.
- `CompleteCheckoutSession` runs DB mutations inside `TxManager.RunInTx`.
- The order row is locked before completion logic reads order status.
- If any DB write fails, no partial completion state commits.
- If the order is already completed, the method returns success without creating a new payment method, payment, or publish event.
- `order.completed` is published only after commit.
- A publish failure does not retry `complete-order`.
- Permanent business errors are non-retryable in Hatchet.

## Tests

Unit tests:

- `PaymentSuccessRunKey` format and uniqueness.
- `PaymentRefundedRunKey` format and uniqueness.
- `CompleteCheckoutSession` uses transaction manager.
- Transaction failure prevents publish.
- Already-completed order is an idempotent no-op.
- Publish happens after successful transaction.

Storage tests:

- `FindByIdForUpdate` returns the same domain order as `FindById`.
- Missing row returns or wraps `port.ErrNotFound`.

Focused commands:

```bash
GOCACHE=/private/tmp/gphq-go-build-cache go test ./internal/adapter/hatchet/workflows
GOCACHE=/private/tmp/gphq-go-build-cache go test ./internal/core/service -run 'TestOrderWorkflowService|TestOrderService_CompleteOrder'
```

