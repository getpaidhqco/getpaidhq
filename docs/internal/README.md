# Internal engineering notes

> **Private / internal docs.** These are working notes for the gphq-server team — local-dev
> setup, debugging war-stories, and deep dives into how the workflow engine actually behaves.
> They are intentionally more candid and lower-level than the public `gphq-docs` site. Not
> customer-facing.

Captured from a live debugging session on **2026-06-03** (local-only environment).

## Index

| Doc | What it covers |
| --- | --- |
| [local-dev-hatchet.md](local-dev-hatchet.md) | Opening the Hatchet UI, the cookie-secret 500 fix, ports |
| [org-seed-data.md](org-seed-data.md) | The `customer_cohorts` FK error, what `OrgService.Create` seeds, the manual-org gotcha, full seed-data audit |
| [hatchet-architecture.md](hatchet-architecture.md) | Mental model: workers / workflows / runs / steps / events, and how a subscription flows through them |
| [durable-runner-timeouts.md](durable-runner-timeouts.md) | **Load-bearing.** Why durable `subscription-runner` runs fail at ~5 min, the execution-timeout model, the annual-subscription problem, and fix options |
| [subscriptions-on-hatchet.md](subscriptions-on-hatchet.md) | **Architecture.** The three execution models (Temporal durable timers / Hatchet long-runner / cron+fan-out), how Temporal's "actor" is cron underneath, Lago's verified production model, where dunning fits, and the A/B/C decision |
| [engine-parity-and-subscription-lifecycle.md](engine-parity-and-subscription-lifecycle.md) | **Load-bearing mental model.** Engine parity = same *behaviour*, not same *code* (two deliberately opposite models); and the subscription first-charge → renewal lifecycle documented per engine (Temporal long-lived actor vs Hatchet cron+fan-out) |
| [plan-cron-fanout-billing.md](plan-cron-fanout-billing.md) | **Implementation plan.** Task-by-task TDD plan to switch Hatchet billing to the cron → per-org fan-out → fresh-per-renewal model (the chosen direction). Executable via superpowers:executing-plans. |
| [logging.md](logging.md) | Why raw GORM SQL logs look "unformatted" next to the zap JSON logs |
| [run-metadata.md](run-metadata.md) | How runs are named, why you can't tell which subscription a run is, and the `WithRunMetadata` fix |

## TL;DR of the session

1. **Hatchet UI 500 on login** → cookie secrets were 7-byte strings; AES needs 16/24/32. Fixed in `docker/docker-compose.yml`.
2. **`customer_cohorts` FK violation** → the org was inserted **manually**, bypassing `OrgService.Create`, so its `signup_date` cohort was never seeded. Seeded the missing row.
3. **Failed `subscription-runner` runs (~5 min each)** → the durable runner sets **no execution timeout**, so Hatchet's **5-minute default** reaps it mid-sleep. Durable waits are **not** exempt from the execution timeout. **Annual subscriptions make this unfixable by a simple timeout bump** — see [durable-runner-timeouts.md](durable-runner-timeouts.md).
4. **Run metadata** added so runs are filterable by org / subscription / campaign in the UI.
