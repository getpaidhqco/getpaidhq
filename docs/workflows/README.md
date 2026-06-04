---
title: Workflows & Flows
description: Mermaid diagrams of every durable workflow and event-driven flow in the GetPaidHQ billing backend.
---

# Workflows & Flows

Diagrams generated from the actual code. Start with **Subscription Payments** for the end-to-end picture, then drill into each durable workflow.

> Architecture overview: [System Architecture (Hexagonal)](../architecture/system-hexagonal.md)

## Subscription billing & payments

- [Subscription Payments — End to End](./subscription-payments.md) — Full path of a subscription payment in GetPaidHQ: order paid, payment-success DAG, durable subscription-runner, billing-cycle charge, and the failure-to-dunning recovery branch.
- [Subscription Runner (Durable Lifecycle)](./subscription-runner.md) — The per-subscription durable workflow that schedules charges, reacts to pause/resume/cancel signals, and drives the subscription lifecycle to a terminal state.
- [Billing Cycle (Charge a Period)](./billing-cycle.md) — How the billing-cycle DAG charges a subscription for one period: compute amount, charge the gateway, record payment, advance billing date, branch to dunning on failure.
- [Payment Success Workflow](./payment-success.md) — The payment-success DAG that completes a paid order and spawns the subscription runner, with parity across the Hatchet and Temporal engines.
- [Payment Refunded Workflow](./payment-refunded.md) — How GetPaidHQ processes a PSP refund event: flip the payment to refunded and record a refund row, with retry parity across Hatchet and Temporal.
- [Dunning & Payment Recovery](./dunning-recovery.md) — How GetPaidHQ recovers failed subscription charges: campaigns, the durable runner's two-phase retry schedule, attempt outcomes, and escalation.
- [Subscription Charge Reminder](./charge-reminder.md) — A durable, timer-driven workflow that sleeps until one minute before a subscription's next charge and publishes a renewal reminder event.
- [Outgoing Webhooks (Delivery)](./outgoing-webhooks.md) — How GetPaidHQ matches domain events to webhook subscriptions and delivers signed, SSRF-guarded HTTP POSTs with Hatchet-managed retries.

## Cross-cutting flows

- [Event-Driven Bridges (NATS Pub/Sub)](./event-bridges.md) — How GetPaidHQ fans domain events over NATS into workflow engines and the reporting DB via SubscriptionEventBridge, DunningOrchestrationService, and ReportEventBridge.
- [Workflow Engine Abstraction (Hatchet ⇄ Temporal)](./workflow-engine-abstraction.md) — How WORKFLOW_ENGINE selects Hatchet or Temporal at boot, both satisfying port.Engine and port.DunningEngine over the same engine-agnostic domain services.
