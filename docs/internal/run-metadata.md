# Run naming & metadata — telling which subscription a run is

## The problem

A run shows in the Hatchet UI as e.g. `subscription-runner-1780504120610`. That name is **not**
informative:

- The **run name is the workflow name** (`"subscription-runner"`); the SDK gives **no option** to
  set a custom display name (`RunOpts` exposes only `AdditionalMetadata` and `Priority` —
  `sdks/go/workflow.go`).
- The `-1780504120610` suffix is an **auto-appended epoch-millisecond timestamp**. Decoded:
  `1780504120610 ms` = `2026-06-03 16:28:40 UTC`. It tells you **when** the run started, nothing
  about **which** subscription.

The identity actually lives in two places:

1. the **run key** `sub_<orgId>_<subscriptionId>` (`workflows/keys.go` `SubscriptionRunKey`), used
   for idempotency, and
2. the **run input payload** (the full `domain.Subscription`).

To find which subscription a run is, you previously had to open the run's **Input** tab, or
cross-reference the app log line `Started subscription-runner RunID=… OrgId=… SubscriptionId=…`
(`hatchet.go`).

## The fix: `WithRunMetadata`

The idiomatic way to make runs **searchable/filterable** in the UI is run metadata (not the name).
The Go SDK helper (`sdks/go/workflow.go:67`) is:

```go
func WithRunMetadata(metadata map[string]string) RunOptFunc
```

It composes with the existing `WithRunKey` — just add it to the same `RunNoWait(...)` / `Run(...)`
call. Values must be `string` (convert ints with `strconv.Itoa`).

## What was added (Hatchet adapter only; Temporal untouched)

| File / function | Workflow | Metadata keys |
| --- | --- | --- |
| `hatchet.go` `StartSubscriptionWorkflow` | `subscription-runner` | `orgId`, `subscriptionId`, `customerId` |
| `hatchet.go` `StartDunningWorkflow` | `dunning-runner` | `orgId`, `campaignId`, `subscriptionId`, `customerId` |
| `hatchet.go` `StartWorkflow` | `payment-success` | `orgId`, `orderId`, `paymentId` |
| `hatchet.go` `StartWorkflow` | `payment-refunded` | `orgId`, `orderId`, `paymentId` |
| `hatchet.go` `StartWorkflow` | `outgoing-webhook` | `orgId`, `webhookSubscriptionId`, `eventId` |
| `workflows/subscription_runner.go` | `subscription-charge-reminder` spawn | `orgId`, `subscriptionId` |
| `workflows/subscription_runner.go` | `billing-cycle` spawn | `orgId`, `subscriptionId`, `cycle` |
| `workflows/dunning_runner.go` | `dunning-attempt` spawn | `orgId`, `campaignId`, `attemptNumber` |

Keys are camelCase and consistent across sites. Run keys and control flow were **not** changed —
metadata is purely additive. Verified with `go build ./...`.

> Field names were checked against the domain types (`domain.Subscription`,
> `domain.PaymentWebhookContext`, `domain.WebhookSubscription`, `domain.StartDunningWorkflowInput`).
> If those structs change, update the metadata sources accordingly.

## Example

```go
ref, err := h.client.RunNoWait(ctx, "subscription-runner", sub,
    hatchet.WithRunKey(hatchetwf.SubscriptionRunKey(sub.OrgId, sub.Id)),
    hatchet.WithRunMetadata(map[string]string{
        "orgId":          sub.OrgId,
        "subscriptionId": sub.Id,
        "customerId":     sub.CustomerId,
    }),
)
```

In the UI you can now filter runs by `orgId` / `subscriptionId` / `campaignId` instead of decoding
timestamps.
