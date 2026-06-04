---
title: Subscription Charge Reminder
description: A durable, timer-driven workflow that sleeps until one minute before a subscription's next charge and publishes a renewal reminder event.
---

# Subscription Charge Reminder

The Subscription Charge Reminder is a short-lived, durable timer workflow that notifies a customer ahead of an upcoming charge. The per-subscription runner spawns one reminder per billing cycle, detached, scheduled for one minute before the next charge date. The reminder durably sleeps until that moment (surviving worker restarts) and then asks `SubscriptionService.SendRenewalReminder` to publish a `subscription.renewal_reminder` pub/sub event. It runs on both engines: a Hatchet durable standalone task and an equivalent Temporal child workflow.

Note: the reminder itself has no signal/cancellation handler. "Cancellation when the subscription changes" is handled by the *parent* runner via spawn idempotency — a stale reminder simply fires against fresh subscription state re-read at send time, and a new reminder is only spawned when the next charge date moves to a new day (the run key/workflow ID is keyed on the `YYYYMMDD` reminder date).

```mermaid
sequenceDiagram
    autonumber
    participant Runner as "Subscription Runner (parent)"
    participant Reminder as "subscription-charge-reminder (durable timer)"
    participant Svc as "SubscriptionService.SendRenewalReminder"
    participant Repo as "SubscriptionRepository"
    participant PubSub as "PubSub"

    Note over Runner: next = sub.GetNextChargeDate()<br/>reminderAt = next - 1 minute
    Runner->>Reminder: spawn detached<br/>Hatchet RunNoWait, run key reminder_org_sub_YYYYMMDD<br/>Temporal child, PARENT_CLOSE_POLICY_ABANDON
    Note over Runner: parent does NOT block;<br/>continues to WaitFor charge time / control events

    activate Reminder
    Note over Reminder: now = ctx.Now()<br/>wait = ReminderAt - now
    alt wait > 0
        Reminder->>Reminder: durable sleep for wait<br/>Hatchet ctx.SleepFor / Temporal temporal.Sleep
    else wait <= 0
        Note over Reminder: skip sleep, send immediately
    end

    Reminder->>Svc: SendRenewalReminder(orgId, subscriptionId)<br/>Temporal via OrderActivities.ProcessReminderEvent
    activate Svc
    Svc->>Repo: FindById(orgId, id) re-read fresh state
    Repo-->>Svc: domain.Subscription
    alt FindById fails
        Repo-->>Svc: error
        Svc-->>Reminder: return err
        Note over Reminder: Hatchet returns err (no retry config)<br/>Temporal wraps NonRetryableApplicationError
    else found
        Svc->>PubSub: Publish orgId, "subscription.renewal_reminder", subscription
        alt Publish fails
            PubSub-->>Svc: error
            Svc-->>Reminder: return err
        else published
            PubSub-->>Svc: ok
            Svc-->>Reminder: nil
        end
    end
    deactivate Svc
    Note over Reminder: Hatchet returns input.Subscription<br/>Temporal returns WorkflowResult{Success:true, Message:"sent"}
    deactivate Reminder
```

## How it works

### Spawn (parent runner)
The reminder is never started on its own; the per-subscription runner spawns it once per loop iteration. In `internal/adapter/hatchet/workflows/subscription_runner.go`, `NewSubscriptionRunnerWorkflow` computes `next := sub.GetNextChargeDate()`, sets `reminderAt := next.Add(-1 * time.Minute)`, and calls `client.RunNoWait(ctx, "subscription-charge-reminder", ReminderInput{...}, hatchet.WithRunKey(ReminderRunKey(...)))` — fire-and-forget, so the parent immediately proceeds to its own `ctx.WaitFor` over the charge time and control events. The Temporal mirror in `internal/adapter/temporal/workflows/subscription_workflow.go` uses `ExecuteChildWorkflow(reminderCtx, SubscriptionChargeReminder, ReminderInput{...})` with `ParentClosePolicy: PARENT_CLOSE_POLICY_ABANDON` and `WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE`, and only waits on `GetChildWorkflowExecution()` to confirm start.

### Idempotency / de-duplication
Both engines key the spawn on the reminder date. `ReminderRunKey` (`internal/adapter/hatchet/workflows/keys.go`) and `ReminderWorkflowID` (`internal/adapter/temporal/workflows/keys.go`) both format as `reminder_<orgId>_<subscriptionId>_<YYYYMMDD>`. A second spawn for the same subscription and same reminder day is de-duplicated, so the parent's wait loop can restart on subscription changes (`subscription.paused/.resumed/.cancelled/.activated`, `refresh-state`) without flooding duplicate reminders. A genuinely new charge date on a different day produces a new key and therefore a new reminder.

### Durable wait
`ReminderInput` (`internal/adapter/hatchet/workflows/types.go`) carries the `Subscription` snapshot and `ReminderAt`. The Hatchet task `NewSubscriptionChargeReminderWorkflow` (`internal/adapter/hatchet/workflows/subscription_charge_reminder.go`) computes `wait := input.ReminderAt.Sub(ctx.Now())` and, when positive, calls `ctx.SleepFor(wait)` — a durable sleep that survives worker restarts. The Temporal `SubscriptionChargeReminder` (`internal/adapter/temporal/workflows/subscription_charge_reminder.go`) does the same with `temporal.Sleep(ctx, wait)`. If `wait <= 0` the sleep is skipped and the reminder sends immediately.

### Send
After the timer, Hatchet calls `subscriptionService.SendRenewalReminder(ctx, input.Subscription.OrgId, input.Subscription.Id)` directly. Temporal routes through the activity `OrderActivities.ProcessReminderEvent` (`internal/adapter/temporal/activities/order_activities.go`), which delegates to the same `SendRenewalReminder`. In `internal/core/service/subscription.go`, `SendRenewalReminder` re-reads the subscription via `subscriptionRepository.FindById(ctx, orgId, id)` to guarantee fresh state, then publishes `port.TopicSubscriptionRenewalReminder` (`"subscription.renewal_reminder"`, `internal/core/port/topic.go`) with the subscription payload.

### Timeouts and error paths
The Hatchet task sets `WithExecutionTimeout(10*time.Second)` covering the send (the durable sleep is separate). The Temporal activity uses `StartToCloseTimeout: 10s` with a `RetryPolicy` of `MaximumAttempts: 1` (no retries). A `FindById` or `Publish` error propagates back: Hatchet returns the error with `input.Subscription`; Temporal wraps it in `temporalio.NewNonRetryableApplicationError("SubscriptionChargeReminder failed", "reminder", err)`. On success Hatchet returns the original `domain.Subscription` and Temporal returns `port.WorkflowResult{Success: true, Message: "sent"}`.
