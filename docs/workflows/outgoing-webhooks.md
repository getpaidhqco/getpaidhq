---
title: Outgoing Webhooks (Delivery)
description: How GetPaidHQ matches domain events to webhook subscriptions and delivers signed, SSRF-guarded HTTP POSTs with Hatchet-managed retries.
---

# Outgoing Webhooks (Delivery)

GetPaidHQ turns internal domain events into outbound HTTP callbacks to tenant-supplied endpoints. A published `PubSubPayload` is matched against the org's webhook subscriptions, and each match fans out into a separate `outgoing-webhook` Hatchet task that signs the body, enforces an SSRF/url-safety guard, and POSTs it. Retries and backoff are owned entirely by the workflow engine; the delivery code only signs and sends. The signing secret produces an HMAC-SHA256 `X-Signature` header so receivers can verify authenticity.

```mermaid
sequenceDiagram
    autonumber
    participant Pub as "PubSub event"
    participant WS as "WorkflowService.HandleOutboundWebhook"
    participant Repo as "WebhookSubscriptionRepository"
    participant Eng as "Engine.StartWorkflow"
    participant HW as "Hatchet task: outgoing-webhook"
    participant Step as "OutgoingWebhookSteps.SendWebhook"
    participant Svc as "WebhookSubscriptionService.SendWebhook"
    participant Guard as "validateOutgoingWebhookURL / safeDialContextWith"
    participant Cust as "Customer endpoint"

    Pub->>WS: "topic, data bytes"
    WS->>WS: "json.Unmarshal into PubSubPayload"
    WS->>Repo: "FindByEvent(orgId, payload.Topic)"
    Repo-->>WS: "[]WebhookSubscription"
    loop "for each matched subscription"
        WS->>Eng: "StartWorkflow(WorkflowOutgoingWebhook, OutgoingWebhookPayload{sub, event})"
        Eng->>HW: "RunNoWait('outgoing-webhook', payload)"
        Note over HW: "WithExecutionTimeout 15s<br/>WithRetries 5<br/>WithRetryBackoff 1.0, 60s"
        HW->>Step: "SendWebhook(ctx, payload)"
        Step->>Svc: "whService.SendWebhook(ctx, payload)"

        Svc->>Guard: "validateOutgoingWebhookURL(url, ipPredicate)"
        alt "URL resolves to internal IP / bad scheme / DNS fail"
            Guard-->>Svc: "ErrUnsafeWebhookURL"
            Svc-->>HW: "return err"
            Note over HW: "attempt fails -> Hatchet retry"
        else "URL is public unicast"
            Svc->>Svc: "json.Marshal(event)"
            Svc->>Svc: "X-Signature = HMAC-SHA256(body, secret) if secret != ''"
            Svc->>Svc: "set Content-Type + X-Timestamp (RFC3339)"
            Svc->>Guard: "httpClient.Do -> safe DialContext re-checks resolved IP"
            alt "DialContext rejects IP (DNS rebinding)"
                Guard-->>Svc: "ErrUnsafeWebhookURL"
                Svc-->>HW: "return err -> retry"
            else "dial allowed"
                Svc->>Cust: "POST body (30s client timeout)"
                Cust-->>Svc: "HTTP response"
                Svc->>Svc: "drain <=64KB, log status, return nil"
                Svc-->>HW: "success"
            end
        end
    end
    Note over HW: "Up to 5 attempts; status code is logged only,<br/>NOT used to decide success. Retry is driven by<br/>returned error (url-safety / transport failures)."
```

## How it works

### Event matching and fan-out
`WorkflowService.HandleOutboundWebhook` in `internal/core/service/workflow.go` is the entry point. It unmarshals the raw bytes into a `port.PubSubPayload`, then calls `whsRepo.FindByEvent(ctx, payload.OrgId, payload.Topic)` to load every `WebhookSubscription` for that org subscribed to the topic. For each match it calls `engine.StartWorkflow(ctx, port.WorkflowOutgoingWebhook, port.OutgoingWebhookPayload{WebhookSubscription: sub, Event: payload})` — one workflow run per subscription. A `StartWorkflow` failure is logged and the loop continues; it does not abort sibling deliveries.

### Workflow dispatch
`Hatchet.StartWorkflow` (`internal/adapter/hatchet/hatchet.go`) handles the `port.WorkflowOutgoingWebhook` case by type-asserting the payload to `port.OutgoingWebhookPayload` and calling `client.RunNoWait(ctx, "outgoing-webhook", wh)`. A bad payload type returns a `portError`; otherwise it returns `WorkflowResult{Success: true, Message: "outgoing-webhook queued"}`. (The Temporal adapter mirrors this under `internal/adapter/temporal/`.)

### The Hatchet task and retry policy
`NewOutgoingWebhookWorkflow` in `internal/adapter/hatchet/workflows/outgoing_webhook.go` registers the standalone task named `"outgoing-webhook"`. The handler calls `whSteps.SendWebhook` and maps a nil error to `WorkflowResult{Success: true, Message: "sent"}`. Retry/backoff is declared on the task itself:

- `hatchet.WithExecutionTimeout(15 * time.Second)`
- `hatchet.WithRetries(5)`
- `hatchet.WithRetryBackoff(1.0, 60)` — backoff coefficient `1.0` (flat) with a 60s interval, i.e. up to 5 attempts

Because retry is engine-owned, any error returned from the step triggers a retry; a nil return marks the attempt successful.

### Delivery step
`OutgoingWebhookSteps.SendWebhook` (`internal/adapter/hatchet/steps/outgoing_webhook_steps.go`) is a thin coordinator: it logs, calls `whService.SendWebhook(ctx, data)`, and propagates any error so Hatchet can retry.

### Signing and HTTP send
`WebhookSubscriptionService.SendWebhook` in `internal/core/service/webhook_subscription.go` performs the actual delivery:

1. Re-validates the destination via `validateOutgoingWebhookURL(ctx, webhook.URL, s.ipPredicate)` before doing any work.
2. `json.Marshal(input.Event)` to produce the request body.
3. Builds a `POST` with `http.NewRequestWithContext`. If `webhook.Secret != ""`, sets `X-Signature` to `generateHMACSignature(jsonData, webhook.Secret)` — `hmac.New(sha256.New, secret)` hex-encoded. Always sets `Content-Type: application/json` and `X-Timestamp` (UTC RFC3339).
4. Sends with the shared `httpClient` (hard `outgoingWebhookTimeout` of 30s). On `Do` error it logs and returns the error (→ retry).
5. On success it drains the response body capped at `outgoingWebhookMaxResponseBytes` (64KB) via `io.LimitReader` so the connection returns to the pool without letting a hostile endpoint exhaust memory, then closes it.

Note: the response status code is only logged (`"webhook delivered"`), not used to determine success — the caller owns retry policy. A delivery returns `nil` (success) as long as the transport completed.

### URL-safety / SSRF guard
`internal/core/service/webhook_url_safety.go` provides defense in depth:

- `validateOutgoingWebhookURL` parses the URL, requires `http`/`https` scheme and a non-empty host, resolves it with `net.DefaultResolver.LookupIP`, and rejects the delivery with `ErrUnsafeWebhookURL` if any resolved IP fails the predicate. This runs both at subscription `Create` time and at every `SendWebhook`.
- `isPublicUnicast` is the production predicate. It rejects loopback, link-local (including the cloud metadata IP `169.254.169.254`), private RFC1918/RFC4193 ranges, multicast/broadcast/unspecified, CGNAT `100.64.0.0/10`, and the IPv4 broadcast address.
- `safeDialContextWith` is wired into the `http.Transport.DialContext` in the service constructor. It re-resolves and re-checks the IP at TCP connect time (DNS-rebinding defense), dials only allowed IPs by their resolved address, and returns `ErrUnsafeWebhookURL` for any disallowed IP. The predicate is read through a getter so tests can swap in `allowAllIPs` — that predicate is package-private and never referenced from production paths.

### Idempotency
Each matched subscription gets its own `outgoing-webhook` run, and Hatchet de-duplicates/retries that run under its `WithRetries(5)` policy, so a transient failure re-attempts the same signed payload rather than re-fanning the event. (The related PSP-inbound idempotency-key claim/release model lives in `WebhookService.HandlePaymentWebhook` in `internal/core/service/webhook.go`, which claims a SHA-256 key before side effects and releases it on downstream failure so the PSP's retry can re-run the work.)
