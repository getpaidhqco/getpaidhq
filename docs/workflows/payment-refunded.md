---
title: Payment Refunded Workflow
description: How GetPaidHQ processes a PSP refund event: flip the payment to refunded and record a refund row, with retry parity across Hatchet and Temporal.
---

# Payment Refunded Workflow

This workflow handles a single refund event coming from a payment service provider (PSP). It is a thin, single-step workflow: it looks up the original payment by its PSP id, flips that payment's status to `refunded`, and writes a `Refund` row. The Hatchet and Temporal adapters both delegate to the same `PaymentService.ProcessRefund` so behaviour stays identical across engines.

Note that, unlike the payment-success path, the refund path publishes **no** NATS events, fires **no** webhooks, and touches **no** subscription or order state — it only mutates the payment and creates the refund record.

```mermaid
sequenceDiagram
    autonumber
    participant Engine as "Workflow Engine (Hatchet / Temporal)"
    participant WF as "payment-refunded task / PaymentRefunded"
    participant Act as "OrderActivities.HandlePaymentRefundedEvent (Temporal only)"
    participant Svc as "PaymentService.ProcessRefund"
    participant Repo as "PaymentRepository"
    participant DB as "Postgres"

    Engine->>WF: trigger with PaymentRefundedInput / PaymentWebhookContext
    Note over WF: Hatchet calls ProcessRefund directly<br/>Temporal calls it via an activity
    WF->>Act: ExecuteActivity HandlePaymentRefundedEvent (Temporal)
    Act->>Svc: ProcessRefund(ctx, paymentContext)

    Svc->>Repo: FindByPspId(orgId, Payment.PspId)
    Repo->>DB: SELECT payment
    alt payment not found / lookup error
        Repo-->>Svc: error
        Svc-->>WF: return error -> engine retries
    else found
        Repo-->>Svc: domain.Payment
        Svc->>Svc: payment.Status = PaymentStatusRefunded
        Svc->>Repo: Update(payment)
        Repo->>DB: UPDATE payment SET status='refunded'
        alt update error
            Repo-->>Svc: error
            Svc-->>WF: return error -> engine retries
        else updated
            Repo-->>Svc: newPayment
            Svc->>Repo: CreateRefund(Refund{Amount, Currency, RefundedAt...})
            Repo->>DB: INSERT refund
            alt create-refund error
                Repo-->>Svc: error
                Svc-->>WF: return error -> engine retries
            else created
                Repo-->>Svc: ok
                Svc-->>WF: newPayment
                WF-->>Engine: success (Temporal: WorkflowResult{Success:true})
            end
        end
    end
```

## How it works

1. **Entry points.** Both engines take a `domain.PaymentWebhookContext` as input. The Hatchet adapter defines a `StandaloneTask` named `"payment-refunded"` whose handler receives `PaymentRefundedInput` and calls `paymentService.ProcessRefund(ctx, input.PaymentContext)` directly — see `internal/adapter/hatchet/workflows/payment_refunded.go` and the input struct in `internal/adapter/hatchet/workflows/types.go`. The Temporal adapter's `PaymentRefunded` workflow instead schedules a single activity, `OrderActivities.HandlePaymentRefundedEvent`, via `ExecuteActivity` — see `internal/adapter/temporal/workflows/payment_refunded.go`.

2. **Activity delegation (Temporal).** `HandlePaymentRefundedEvent` in `internal/adapter/temporal/activities/order_activities.go` is a thin coordinator: it logs, calls `a.paymentService.ProcessRefund(ctx, paymentContext)`, and on error wraps it with `temporal.NewApplicationError("Can't process refund", "refund", err)` — a **retryable** application error (unlike the order-completion path, which uses `NewNonRetryableApplicationError`).

3. **Core logic.** `PaymentService.ProcessRefund` in `internal/core/service/payment.go` does the actual work:
   - `paymentRepository.FindByPspId(ctx, paymentContext.OrgId, paymentContext.Payment.PspId)` locates the original payment by org + PSP id.
   - Sets `payment.Status = domain.PaymentStatusRefunded` (the `"refunded"` enum from `internal/core/domain/payment_types.go`) and persists it with `paymentRepository.Update`.
   - Calls `paymentRepository.CreateRefund` with a new `domain.Refund` (`internal/core/domain/refund.go`), id generated via `lib.GenerateId("refund")`, copying `Amount` and `Currency` from `paymentContext.Payment` and stamping `RefundedAt`/`CreatedAt`/`UpdatedAt` with `time.Now().UTC()`.
   - Returns the updated payment. Any repository error is logged and returned, which bubbles up to the engine and triggers a retry.

4. **Retry / idempotency.** Both engines cap at **10 attempts** with a flat backoff. Hatchet: `WithExecutionTimeout(10s)`, `WithRetries(10)`, `WithRetryBackoff(1.0, 60)` (the source comment claiming "indefinitely / no max-attempts" is stale — the code sets 10). Temporal: `StartToCloseTimeout: 10s`, `RetryPolicy{InitialInterval: 1m, BackoffCoefficient: 1.0, MaximumAttempts: 10}`. There is no explicit idempotency guard; a re-delivered refund event would re-set the status to `refunded` and insert another `Refund` row.

5. **Result.** Temporal returns `port.WorkflowResult{Success: true, Message: "Refund event processed", Payload: payment}`; Hatchet returns the `domain.Payment` directly. No events, webhooks, or subscription side effects are emitted in this path.
