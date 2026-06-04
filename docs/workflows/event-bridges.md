---
title: Event-Driven Bridges (NATS Pub/Sub)
description: How GetPaidHQ fans domain events over NATS into workflow engines and the reporting DB via SubscriptionEventBridge, DunningOrchestrationService, and ReportEventBridge.
---

# Event-Driven Bridges (NATS Pub/Sub)

GetPaidHQ decouples the domain layer from the workflow engine and the reporting database with a set of NATS pub/sub *bridges*. Domain services publish `subscription.*`, `payment.*`, `customer.*` and `refund.*` events through `port.PubSub`; three subscribers — `SubscriptionEventBridge`, `DunningOrchestrationService` and `ReportEventBridge` — consume them and translate each into a side effect (forwarding to the engine, starting a dunning workflow, or upserting a reporting row). Every subscriber is wrapped in `safePubSubHandler` so a single panicking callback can never take down the shared NATS receive loop.

## Flow: topics to subscribers to actions

```mermaid
flowchart TD
    subgraph Publishers["Publishers — SubscriptionService (internal/core/service/subscription.go)"]
        P1["Publish: subscription.* lifecycle<br/>created / paused / activated / resumed /<br/>cancelled / unpaid / expired / completed /<br/>past_due / billing_anchor_changed"]
        P2["Publish: subscription.payment.charge.failed<br/>map{subscription, charge_result}"]
        P3["Publish: subscription.payment.charge.success<br/>SubscriptionPaymentChargeSuccessEvent"]
        P4["Publish: payment.created / payment.updated /<br/>payment.failed"]
        P5["Publish: customer.created"]
    end

    subgraph Subjects["NATS subjects (subscribed patterns)"]
        S1["subscription.*"]
        S2["subscription.payment.charge.failed"]
        S3["subscription.>"]
        S4["payment.>"]
        S5["customer.>"]
        S6["refund.>"]
    end

    P1 --> S1
    P1 --> S3
    P2 --> S2
    P2 --> S3
    P3 --> S3
    P4 --> S4
    P5 --> S5

    subgraph Bridges["Subscribers (all wrapped by safePubSubHandler)"]
        B1["SubscriptionEventBridge.Handle"]
        B2["DunningOrchestrationService.HandleSubscriptionChargeFailure"]
        B3["ReportEventBridge.Handle"]
    end

    S1 --> B1
    S2 --> B2
    S3 --> B3
    S4 --> B3
    S5 --> B3
    S6 --> B3

    B1 -->|"topic == subscription.paused only;<br/>all others dropped"| A1["engine.UpdateSubscriptionWorkflow<br/>updateName=topic, subscription"]
    B2 --> A2["StartDunningWorkflow<br/>CreateCampaign + dunningEngine.StartDunningWorkflow<br/>persist WorkflowId / WorkflowRunId"]
    B3 -->|"subscription lifecycle topics"| A3["reportRepo.UpsertSubscription"]
    B3 -->|"subscription.payment.charge.success"| A4["reportRepo.UpsertPayment(evt.Payment)"]
    B3 -->|"payment.created / updated / failed"| A5["reportRepo.UpsertPayment"]
    B3 -->|"customer.created"| A6["reportRepo.UpsertCustomer"]
    B3 -->|"payment_method.*, subscription.workflow.*,<br/>renewal_reminder, refund.*"| A7["default: no-op (unrouted)"]
```

## How it works

### Publishers

All operational events originate in domain services, primarily `SubscriptionService` in `internal/core/service/subscription.go`. Lifecycle transitions are published via `port.GetSubscriptionTopic(status)` (mapping `SubscriptionStatus` to e.g. `subscription.activated`, `subscription.paused`). Charge outcomes publish two distinct shapes:

- **Failure** (`HandleSubscriptionChargeFailure`, line ~584) publishes `subscription.payment.charge.failed` carrying a `map[string]any{"subscription": ..., "charge_result": ...}`, then `payment.created`, then a status-specific lifecycle topic (`cancelled` / `unpaid` / `expired`, or `past_due` only when `subscription.Retries == 1`).
- **Success** (line ~499) publishes `subscription.payment.charge.success` carrying a `port.SubscriptionPaymentChargeSuccessEvent` built by `NewSubscriptionPaymentChargeSuccessEvent`, which embeds the full `domain.Payment`.

Topic strings are the single source of truth in `internal/core/port/topic.go`.

### `SubscriptionEventBridge` — engine fan-in

`NewSubscriptionEventBridge` (`internal/core/service/subscription_event_bridge.go`) subscribes to the wildcard `subscription.*` and registers `Handle` through `safePubSubHandler`. `Handle` unmarshals the `port.PubSubPayload` envelope, re-marshals `envelope.Data`, and decodes it into a `domain.Subscription`. It then switches on `topic`: **only** `port.TopicSubscriptionPaused` (`subscription.paused`) is forwarded — it calls `engine.UpdateSubscriptionWorkflow(ctx, topic, sub)` (`internal/core/port/workflow.go`), passing the topic string as the update name on the per-subscription durable runner. Every other `subscription.*` topic falls through to the `default` branch and is logged and dropped, since the engine has no observer for it.

### `DunningOrchestrationService` — auto-start dunning

`NewDunningOrchestrationService` (`internal/core/service/dunning_orchestration.go`) subscribes to the exact subject `port.TopicSubscriptionPaymentChargeFailed` (`subscription.payment.charge.failed`) and registers `HandleSubscriptionChargeFailure` via `safePubSubHandler`. The handler decodes the envelope into the publisher's `{subscription, charge_result}` shape, then calls `StartDunningWorkflow` with a `domain.StartDunningWorkflowInput` populated from `payload.Subscription` and `payload.ChargeResult` (failed amount, currency, error reason/code, plus `metadata["triggered_by"] = "subscription_charge_failure"`). `StartDunningWorkflow` resolves dunning config (falling back to `domain.DefaultDunningConfig()` on error), snapshots it onto the campaign via `CreateCampaign`, calls `dunningEngine.StartDunningWorkflow` (`internal/core/port/dunning.go`) to obtain `(workflowId, runId)`, and persists those handles back onto the campaign through `UpdateCampaign`. Failures are logged and reported via `errorReporter.ReportError`; the event is not retried.

### `ReportEventBridge` — reporting upserts

`NewReportEventBridge` (`internal/core/service/report_event_bridge.go`) subscribes one wrapped handler to four recursive wildcards: `subscription.>`, `payment.>`, `customer.>`, `refund.>`. `Handle` decodes the envelope, then routes by `topic`:

- Subscription lifecycle topics (`created`, `paused`, `activated`, `resumed`, `cancelled`, `unpaid`, `expired`, `completed`, `past_due`, `billing_anchor_changed`) decode to `domain.Subscription` and call `reportRepo.UpsertSubscription`.
- `subscription.payment.charge.success` decodes to `port.SubscriptionPaymentChargeSuccessEvent` and upserts `evt.Payment` via `reportRepo.UpsertPayment`.
- `payment.created` / `payment.updated` / `payment.failed` decode to `domain.Payment` and call `reportRepo.UpsertPayment`.
- `customer.created` decodes to `domain.Customer` and calls `reportRepo.UpsertCustomer`.
- Everything else in the subscribed namespaces (`payment_method.*`, `subscription.workflow.*`, `subscription.renewal_reminder`, and all `refund.>`) hits the `default` branch and is intentionally ignored — no reporting table exists for them yet.

Each upsert is independent and idempotent; a missed event self-heals on the next event for that entity, and the nightly `ProcessDailyMetrics` cron aggregates the resulting rows.

### Panic safety

`safePubSubHandler` (`internal/core/service/pubsub_handler.go`) wraps every `Subscribe` callback in a `recover()` that logs the handler name, topic, recovered value and stack trace, and deliberately does **not** re-raise. Dropping one event is preferred over crashing the shared NATS receive loop and silencing all other subscribers. Bridge constructors also return an `error` from the first failed `Subscribe` rather than panicking, so a transient NATS hiccup during boot no longer crashes the process before the HTTP server is up.
