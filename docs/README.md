---
title: GetPaidHQ Documentation
description: Architecture and workflow diagrams for the GetPaidHQ subscription-billing backend, generated from the code.
---

# GetPaidHQ Documentation

Diagrams and explanations of how the GetPaidHQ billing backend is built and how its money-movement flows work. Every diagram is derived from the actual source and cites the files it came from.

## Architecture

- [System Architecture (Hexagonal / Ports & Adapters)](./architecture/system-hexagonal.md) — the hexagon: domain + services at the center, ports as interfaces, adapters depending inward only.

## Workflows & flows

Start with **Subscription Payments** for the end-to-end picture, then drill into each durable workflow. Full index: [docs/workflows](./workflows/README.md).

### Subscription billing & payments
- [Subscription Payments — End to End](./workflows/subscription-payments.md) — the flagship trace: order paid → payment-success → durable runner → billing-cycle charge → failure→dunning.
- [Subscription Runner (Durable Lifecycle)](./workflows/subscription-runner.md)
- [Billing Cycle (Charge a Period)](./workflows/billing-cycle.md)
- [Payment Success Workflow](./workflows/payment-success.md)
- [Payment Refunded Workflow](./workflows/payment-refunded.md)
- [Dunning & Payment Recovery](./workflows/dunning-recovery.md)
- [Subscription Charge Reminder](./workflows/charge-reminder.md)
- [Outgoing Webhooks (Delivery)](./workflows/outgoing-webhooks.md)

### Cross-cutting flows
- [Event-Driven Bridges (NATS Pub/Sub)](./workflows/event-bridges.md)
- [Workflow Engine Abstraction (Hatchet ⇄ Temporal)](./workflows/workflow-engine-abstraction.md)

---

Diagrams are [Mermaid](https://mermaid.js.org/) and render on GitHub and most Markdown viewers.
