# Hexagonal Domain Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Strip every GORM tag, every persistence concern, and every wire-format concern (JSON tags) from `internal/core/domain/`. Move GORM column/relationship/serializer tags to a new `*Row` row-type layer in `internal/adapter/postgres/`. Move JSON tags and API response shapes to DTOs in `internal/adapter/http/`. Connect the layers with explicit `toDomain` / `fromDomain` mapper functions.

**Architecture:** Canonical hexagonal — domain + application + ports + adapters. Each layer has exactly one responsibility, and the line between them is enforced by what tags / imports / types are allowed there.

```
internal/adapter/http/        ┌─ <Entity>Request DTOs   (validate + json tags)
                              ├─ <Entity>Response DTOs  (json tags only)
                              ├─ (req).ToInput(orgId)   <- request DTO → service input
                              └─ NewXxxFromEntity(...)  <- domain entity → response DTO

internal/core/service/        ┌─ Application services (use cases)
                              └─ Command/Query INPUT types  (NO tags, plain Go structs)
                                  e.g. service.CreateSubscriptionInput

internal/core/domain/         ─ Entities, value objects, domain services, domain events
                              ─ NO tags whatsoever (no gorm, no json, no validate)
                              ─ Aggregates reference others by ID, not embedded struct
                              ─ NO command/input types — those are use-case concerns

internal/core/port/           ─ Port interfaces only (Repository, Engine, GatewayAdapter, ...)

internal/adapter/postgres/    ┌─ <entity>Row types (gorm tags only)
                              ├─ (row).toDomain() <Entity>     <- row → domain
                              └─ <entity>RowFromDomain(...)    <- domain → row
```

**Litmus test for what lives where:** *Would this type still make sense if there were no use cases?* Aggregates (Subscription, Order, ...) and value objects (Address, Money, ...) survive that test — they belong in `domain/`. Command/input types (CreateSubscriptionInput, ...) and **Read Models** (OrderDetails, SubscriptionDetails — the composed results of a named query) do not — they belong in `service/`.

**Composed responses (CQRS-flavored DDD):** When an HTTP response nests related entities (Order with Customer + Items + Prices), the composition lives in `service/` as a **Read Model**, the application service exposes a Query Handler that builds it, and the HTTP DTO mapper consumes the read model. Repos return single aggregates only.

**Tech Stack:** Go 1.24, GORM v2, Fuego (HTTP), go-playground/validator (request validation), Testcontainers (integration tests).

---

## Background & Discoveries

This plan was scoped after a survey of the existing code. Key findings that shape the work:

1. **No GORM lifecycle hooks** (`BeforeCreate`, `AfterFind`, etc.) exist anywhere in `internal/core/domain/`. That eliminates the hardest migration pitfall. The entanglement is purely declarative: `gorm:"column:..."` tags, `serializer:json` / `serializer:nulltime`, `foreignKey:...;references:...`, and `TableName()` methods.

2. **Partial DTO infrastructure already exists** at `internal/adapter/http/response.go`. It has DTOs + `NewXxxFromEntity` mappers for **11 of 20 entities**: Order, OrderItem, Subscription, Customer, Product, Variant, Price, Payment, ProrationDetails, Gateway (=PspConfig), Cart. The naming convention is `<Entity>Response` for the DTO and `NewXxxFromEntity(domain.X) <Entity>Response` for the mapper. **We extend this pattern; we do not invent a new one.**

3. **DTO use is inconsistent.** `SubscriptionHandler.Get` correctly returns `SubscriptionResponse`, but `SubscriptionHandler.Update` and `Pause` return `domain.Subscription` directly. Every handler will be audited and switched to DTO returns.

4. **No postgres row types exist today.** GORM operates against `domain.Subscription` (etc.) directly. We are introducing the row layer for the first time.

5. **Embedded relationship fields are part of the public API** for 5 of 7 cases (`Order.Customer`, `Order.Items`, `OrderItem.Price`, `Product.Variants`, `Variant.Prices`). The DTO layer must continue to nest them in responses, but the **domain** type holds only IDs. Repos that need the relationship populate it via separate queries and return composite results to services.

6. **One domain method reads through a relationship**: `Subscription.SetActivationDates()` uses `s.OrderItem.Price.*`. After the split, `s.OrderItem` doesn't exist on the domain type. **Task 1.1 refactors this method to take a `Price` argument.**

7. **`Subscription.IsDueForBilling` has a documented contract with `SubscriptionRepository.FindDueForBilling`** (the SQL mirror). The same status/date rules must hold — we won't touch the Go logic, only its tags.

8. **Test DB isolation is load-bearing** (per `CLAUDE.md`). All DB-touching tests use `testDB(t)` from `setup_test.go` and never read `DATABASE_URL`. The refactor preserves this — row types are introduced in the same package as the existing tests.

9. **`*Input` types currently live in `domain/`** (32 distinct types, ~119 reference sites). Canonical hexagonal puts command/query DTOs in the application/use-case layer — `internal/core/service/`. They are moved there in **Task 0.3 as a single sweep**, before the per-entity work begins. The 32 types:

   ```
   ActivatePaymentUpdateTokenInput  CancelDunningCampaignInput     CancelSubscriptionInput
   CompleteCheckoutSessionInput     CompleteOrderInput             CreateCustomerInput
   CreateDunningCampaignInput       CreateDunningConfigurationInput
   CreateOrderInput                 CreatePaymentMethodInput       CreatePaymentUpdateTokenInput
   CreatePriceInput                 CreateProductInput             CreateProductPriceInput
   CreateProductVariantInput        CreateSessionInput             CreateSubscriptionInput
   CreateVariantInput               CreateWebhookSubscriptionInput PauseDunningCampaignInput
   PauseSubscriptionInput           ResumeDunningCampaignInput     ResumeSubscriptionInput
   StartDunningWorkflowInput        SubscriptionChargeInput        TriggerManualAttemptInput
   UpdateBillingAnchorInput         UpdateDunningConfigurationInput
   UpdatePaymentMethodInput         UpdateProductInput             UpdateSubscriptionInput
   UpdateVariantInput
   ```

10. **Hatchet/Temporal step inputs are serialized** to a durable log. Moving a Go type's package does NOT change the on-the-wire JSON shape (JSON serialization is field-based, not type-based), so in-flight tasks survive **as long as field names are preserved**. The sweep MUST NOT rename fields. For prod-style deploys: drain workers → deploy → restart. For local dev: irrelevant.

---

## Entity Inventory (full surface area)

| # | Domain file | Entity types (separate `TableName`) | gorm fields | Embedded rels (JSON-visible?) | JSON-serialized | Has DTO today? |
|---|---|---|---:|---|---|---|
| 1 | `subscription.go` | `Subscription` | 33 | `Customer` (no), `OrderItem` (no) | `Metadata` | yes |
| 2 | `dunning.go` | `DunningCampaign`, `DunningAttempt`, `DunningCommunication`, `PaymentUpdateToken`, `DunningConfiguration`, `CustomerDunningHistory` | 118 | none | many (`ConfigSnapshot`, `Metadata`, `ProcessorResponse`, `PersonalizationData`, `ProviderResponse`, `TokenData`, `AllowedActions`, `TargetRules`, `Config`) | no |
| 3 | `customer.go` | `Customer`, `CustomerCohort`, `Cohort` | 26 | none | `BillingAddress`, `Metadata` | yes (Customer); no (Cohort, CustomerCohort) |
| 4 | `price.go` | `Price` | 19 | none | `Metadata` | yes |
| 5 | `payment.go` | `Payment` | 18 | none | `Metadata` | yes |
| 6 | `order_item.go` | `OrderItem` | 16 | `Price` (yes) | `Metadata` | yes |
| 7 | `payment_method.go` | `PaymentMethod` | 14 | none | `BillingAddress`, `Details`, `Metadata` | no |
| 8 | `order.go` | `Order` | 14 | `Customer` (yes), `Items` (yes) | `Metadata` | yes |
| 9 | `refund.go` | `Refund` | 10 | none | none | no |
| 10 | `variant.go` | `Variant` | 9 | `Prices` (yes) | `Metadata` | yes |
| 11 | `product.go` | `Product` | 8 | `Variants` (yes) | `Metadata` | yes |
| 12 | `org.go` | `Org` | 8 | none | `Metadata` | no |
| 13 | `metadata_store.go` | `MetadataStore` | 8 | none | none | no |
| 14 | `cart.go` | `Cart` | 8 | none | `Data`, `Metadata` | yes |
| 15 | `api_key.go` | `ApiKey` | 8 | none | none | no |
| 16 | `webhook_subscription.go` | `WebhookSubscription` | 7 | none | `Events` | no |
| 17 | `setting.go` | `Setting` | 7 | none | `Value` | no |
| 18 | `psp.go` | `PspConfig` | 7 | none | none | yes (`GatewayResponse`) |
| 19 | `session.go` | `Session` | 5 | none | none | no |
| 20 | `user.go` | `User` | 4 | none | none | no |

**Total: 25 persisted entity types across 20 files, ~340 `gorm:""` field tags, 7 relationship tags, ~30 `serializer:json` / `serializer:nulltime` tags.**

---

## File Structure (where things land)

### Created files

For every domain file `X.go`, a sibling row file in postgres:
```
internal/adapter/postgres/<entity>_row.go      <- new for every entity
```

Mapper helpers shared across rows:
```
internal/adapter/postgres/row_helpers.go       <- nulltime conversion, JSON map helpers
```

DTO files in HTTP for entities that don't have one in `response.go` today:
```
internal/adapter/http/api_key_dto.go           <- new
internal/adapter/http/org_dto.go               <- new
internal/adapter/http/refund_dto.go            <- new
internal/adapter/http/payment_method_dto.go    <- new
internal/adapter/http/session_dto.go           <- new
internal/adapter/http/setting_dto.go           <- new
internal/adapter/http/user_dto.go              <- new
internal/adapter/http/webhook_subscription_dto.go  <- new
internal/adapter/http/metadata_store_dto.go    <- new (if a handler exists)
internal/adapter/http/dunning_response_dto.go  <- new (extends dunning_dto.go which is request-only today)
```

A short architectural reference doc:
```
docs/internal/hexagonal-mapping-pattern.md     <- new (15 mins to read)
```

### Modified files

- `internal/core/domain/*.go` — strip all `gorm:""` and `json:""` tags from persisted entity types; remove `TableName()` methods; remove embedded relationship struct fields; refactor `Subscription.SetActivationDates()` to take `Price` param.
- `internal/adapter/postgres/*_repo.go` — every repo uses row types internally. Repos return composite "with related" structs (or accept a hydrate flag) where service code currently consumes embedded relations.
- `internal/adapter/http/*_handler.go` — every handler returns `<Entity>Response` (not `domain.X`). Audit and convert.
- `internal/adapter/http/response.go` — keep the existing DTOs; this file already houses the pattern.
- `internal/core/service/*.go` — touch only when service reads through an embedded relation (e.g. `subscription.OrderItem.Price`). Those call sites change to take the related entity explicitly.
- `internal/core/port/repository.go` — repo signatures change for any repo whose return shape changes (e.g. introducing `OrderWithCustomer`).

### Deleted files

None. We aren't removing capabilities; we're separating concerns.

---

## The Mapping Pattern (memorize this)

Every entity follows this exact pattern. Pilot Task 1 demonstrates it on Subscription. Subsequent tasks reference this section.

### A. Domain (in `internal/core/domain/<entity>.go`)

```go
package domain

import "time"

// <Entity> is the domain entity. Pure Go. No persistence or wire-format
// concerns. Cross-aggregate references are by ID only.
type <Entity> struct {
    OrgId           string
    Id              string
    // ... fields, NO tags except for value-object types where a tag is
    // semantically part of the value (none in this codebase today).

    // NO json tags. NO gorm tags. NO validate tags (validation lives on
    // request DTOs in the HTTP layer).

    // NO embedded relationship fields. Hold only the foreign key.
    CustomerId      string
    // (was: Customer Customer)  -- DELETED

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### B. Row (in `internal/adapter/postgres/<entity>_row.go`)

```go
package postgres

import (
    "time"
    "getpaidhq/internal/core/domain"
)

// <entity>Row is the postgres on-the-wire representation. GORM tags are
// the only annotation. lowercase — internal to the postgres adapter.
type <entity>Row struct {
    OrgId           string             `gorm:"column:org_id;primaryKey"`
    Id              string             `gorm:"column:id;primaryKey"`
    CustomerId      string             `gorm:"column:customer_id"`
    Status          string             `gorm:"column:status"`
    StartDate       time.Time          `gorm:"column:start_date;serializer:nulltime"`
    Metadata        map[string]string  `gorm:"column:metadata;serializer:json"`
    CreatedAt       time.Time          `gorm:"column:created_at"`
    UpdatedAt       time.Time          `gorm:"column:updated_at"`
}

func (<entity>Row) TableName() string { return "<table>" }

// toDomain converts a row to the domain entity.
func (r <entity>Row) toDomain() domain.<Entity> {
    return domain.<Entity>{
        OrgId:      r.OrgId,
        Id:         r.Id,
        CustomerId: r.CustomerId,
        Status:     domain.<EntityStatus>(r.Status), // re-typed if needed
        StartDate:  r.StartDate,
        Metadata:   r.Metadata,
        CreatedAt:  r.CreatedAt,
        UpdatedAt:  r.UpdatedAt,
    }
}

// <entity>RowFromDomain converts a domain entity to its row form.
func <entity>RowFromDomain(e domain.<Entity>) <entity>Row {
    return <entity>Row{
        OrgId:      e.OrgId,
        Id:         e.Id,
        CustomerId: e.CustomerId,
        Status:     string(e.Status),
        StartDate:  e.StartDate,
        Metadata:   e.Metadata,
        CreatedAt:  e.CreatedAt,
        UpdatedAt:  e.UpdatedAt,
    }
}
```

### C. Repo update pattern

```go
// BEFORE:
func (r *Repo) FindById(ctx context.Context, orgId, id string) (domain.Entity, error) {
    var e domain.Entity
    err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&e).Error
    return e, translateErr(err)
}

// AFTER:
func (r *Repo) FindById(ctx context.Context, orgId, id string) (domain.Entity, error) {
    var row entityRow
    err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&row).Error
    if err != nil {
        return domain.Entity{}, translateErr(err)
    }
    return row.toDomain(), nil
}

// BEFORE:
func (r *Repo) Create(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
    err := dbFromCtx(ctx, r.db).Create(&entity).Error
    if err != nil { return domain.Entity{}, err }
    return r.FindById(ctx, entity.OrgId, entity.Id)
}

// AFTER:
func (r *Repo) Create(ctx context.Context, entity domain.Entity) (domain.Entity, error) {
    row := entityRowFromDomain(entity)
    err := dbFromCtx(ctx, r.db).Create(&row).Error
    if err != nil { return domain.Entity{}, err }
    return r.FindById(ctx, entity.OrgId, entity.Id)
}
```

### D. Handler / DTO pattern

```go
// In internal/adapter/http/<entity>_handler.go — every method returns a Response DTO.
func (h *Handler) Get(c fuego.ContextNoBody) (EntityResponse, error) {
    authUser := AuthUserFrom(c)
    entity, err := h.service.FindById(c.Context(), authUser.OrgId, c.PathParam("id"))
    if err != nil { return EntityResponse{}, NewApiErrorFromError(err) }
    return NewEntityFromEntity(entity), nil
}
```

### E. Quirks the playbook handles

- **`time.Time` columns that are nullable** keep `serializer:nulltime` on the **row**. Domain holds plain `time.Time` (zero value = unset, same as today).
- **`map[string]string` / `map[string]any` columns** keep `serializer:json` on the row. Domain holds plain `map[...]...`.
- **Value-object columns** like `BillingAddress Address` keep `serializer:json` on the row. Domain holds the value-object by value, unchanged.
- **Embedded slice/struct relations** (e.g. `Order.Items []OrderItem`) are removed from the domain entity. Repos that need them return a composite struct (`OrderWithItems`) or hydrate via a separate query. DTO mappers compose nested responses by accepting the composite.
- **`gorm:"-"` ignored fields** (e.g. `ApiKey.RawKey`, `User.Password`, `Cart.Status`, `Cart.Total`) move to the domain entity as plain fields (no tag needed, since domain has no GORM). They're already not persisted; behaviour is unchanged.
- **`gorm:"-" json:"metadata,omitempty"` fields** (e.g. `CustomerCohort.Metadata`) become plain domain fields. The DTO defines its own `json:"metadata,omitempty"`.

---

## Phase 0 — Scaffolding & Pattern Reference

### Task 0.1: Write the canonical architecture reference doc

**Files:**
- Create: `docs/internal/hexagonal-mapping-pattern.md`

- [ ] **Step 1: Write the full rules doc**

Create `docs/internal/hexagonal-mapping-pattern.md` with this exact content:

````markdown
# Hexagonal & DDD Rules (Payloop server)

This is the rulebook for where types live and what tags they may carry. It is
the canonical reference; if anything in this repo disagrees with it, the file
in this repo is wrong and should be fixed.

## The architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│ Adapters (the outside world)                                             │
│                                                                          │
│  internal/adapter/http/     internal/adapter/postgres/                   │
│   • Request DTOs             • <entity>Row types                         │
│   • Response DTOs            • Row ↔ domain mappers                      │
│   • Request → Input maps     • Repo implementations                      │
│   • Domain → Response maps                                               │
│                                                                          │
│  internal/adapter/hatchet/  internal/adapter/temporal/   ... others ...  │
│   • Workflow & step shims    • Workflow & activity shims                 │
└──────────────────────────────┬───────────────────────────────────────────┘
                               │ talks to core ONLY via port interfaces
┌──────────────────────────────┴───────────────────────────────────────────┐
│ The Core (the hexagon)                                                   │
│                                                                          │
│  internal/core/port/      ─ port interfaces only                         │
│                                                                          │
│  internal/core/service/   ─ APPLICATION layer                            │
│                            • Application services (use cases)            │
│                            • Command/Query INPUT types                   │
│                                                                          │
│  internal/core/domain/    ─ DOMAIN layer                                 │
│                            • Aggregates (entity + value objects)         │
│                            • Domain services (pure business operations)  │
│                            • Domain events                               │
└──────────────────────────────────────────────────────────────────────────┘
```

## What goes where

| Type kind | Lives in | Tags allowed | Examples |
|---|---|---|---|
| Aggregate / entity | `internal/core/domain/` | none | `Subscription`, `Order`, `Customer`, `Payment` |
| Value object | `internal/core/domain/` | none | `Address`, `Pagination`, `ProrationDetails` |
| Enum / status type | `internal/core/domain/` | none | `SubscriptionStatus`, `OrderStatus`, `Currency` |
| Domain service | `internal/core/domain/` | none | `domain.PaymentCalculator` (pure function or method) |
| Domain event | `internal/core/domain/` | none | `OrderCompletedEvent` |
| Command / Query Input | `internal/core/service/` | none | `service.CreateSubscriptionInput`, `service.PauseSubscriptionInput` |
| **Read Model** (composed query result) | `internal/core/service/` | none | `service.OrderDetails`, `service.SubscriptionDetails` |
| Port interface | `internal/core/port/` | none | `port.Repository`, `port.Engine` |
| HTTP Request DTO | `internal/adapter/http/` | `json:""`, `validate:""` | `CreateSubscriptionRequest` |
| HTTP Response DTO | `internal/adapter/http/` | `json:""` | `SubscriptionResponse` |
| Postgres row | `internal/adapter/postgres/` | `gorm:""` | `subscriptionRow` |

## Domain layer rules

1. **Zero framework tags.** No `gorm:""`, no `json:""`, no `validate:""`. The
   domain layer must compile against a hypothetical world where GORM, the HTTP
   framework, and the validator do not exist.

2. **No persistence concerns.** No `TableName()` methods. No `Preload`
   knowledge. No SQL strings.

3. **No wire-format concerns.** No JSON field naming, no `omitempty`
   considerations.

4. **No command/input types.** A `CreateSubscriptionInput` is a use-case
   concern (it describes how the outside world drives the application). It
   lives in `internal/core/service/`, not here.

5. **Cross-aggregate references are by ID.** A `Subscription` holds
   `CustomerId string`, not an embedded `Customer Customer` field. Loading the
   customer is a use-case concern. Repos may expose composite-fetch helpers
   when the use case demands one.

6. **Domain methods take what they need explicitly.** If `Subscription.SetActive`
   needs a `Price`, the price is a parameter, not something assumed to be
   loaded on `self`. This makes the method independent of how the entity was
   constructed and prevents implicit "the relation must be hydrated" coupling.

7. **Aggregates own their invariants.** Methods that mutate state validate
   internal consistency. They are not passive structs — they enforce the
   business rules.

## Application (service) layer rules

1. **Use cases / application services live here**, one per file usually
   (`subscription.go`, `order.go`, ...). They depend on:
   - Domain types (entities, value objects, domain services)
   - Port interfaces (never concrete adapters)
   - Their own command/input types and Read Models

2. **Command and query INPUT types live here.** Examples:
   - `service.CreateSubscriptionInput`
   - `service.PauseSubscriptionInput`
   - `service.GetOrderQuery` (only when the query needs a named type)

3. **READ MODELS live here.** A read model is the composed result of a named
   query. It exists when the HTTP response (or another adapter's response)
   nests related entities. Examples:
   - `service.OrderDetails { Order domain.Order; Customer domain.Customer; Items []service.OrderItemDetails }`
   - `service.SubscriptionDetails { Subscription domain.Subscription; Customer domain.Customer }`

   Rules for read models:
   - Named after the **query**, not the aggregate (e.g. `OrderDetails` not `OrderView`).
   - Composed of `domain.X` entities (or nested read models). NEVER contain
     adapter types.
   - Have NO tags.
   - Live next to the service that produces them (file: `service/<entity>_read.go`).
   - The application service has a Query Handler method that returns the read
     model: `func (s *OrderService) GetDetails(ctx, orgId, id string) (OrderDetails, error)`.

4. **Input and read model types are passive structs with no tags.**
   Validation already happened at the HTTP boundary; the request DTO carries
   `validate:""` tags and maps to an Input via `.ToInput(...)`.

5. **Application services do NOT import adapters.** If a service needs HTTP
   request shape, that's a sign the request shape is wrong, not that the
   service should import HTTP.

6. **Application services orchestrate; they do not contain business rules.**
   Business rules live on domain methods. The service composes them.

7. **Repositories return aggregate roots only.** A subscription repo returns
   `domain.Subscription`, never `Subscription + Customer`. Composition is the
   application service's job (it calls multiple repos, or calls batched
   variants like `FindByIds`).

## Adapter rules (general)

1. **Adapters depend on the core, not the other way around.** Anything in
   `internal/adapter/` may import `internal/core/...`. Nothing in
   `internal/core/` may import `internal/adapter/...`.

2. **Adapters cross the boundary through mappers.** A repo accepts and returns
   domain entities; a row type is package-internal. A handler accepts and
   returns DTOs; the domain entity is package-internal to it.

## Postgres adapter specifics

- Row types are **lowercase** (`subscriptionRow`) — internal to the package.
- `TableName()` lives on the row.
- GORM relationship tags MAY exist on rows when a composite `Preload` query
  is the cheapest correct shape (e.g. `OrderRow` preloading `Items` and
  `Customer`). The repo method that uses the composite returns either a
  composite DTO local to the package, or a tuple of domain types.
- Mappers: `(r row) toDomain() domain.Entity` and
  `entityRowFromDomain(e domain.Entity) row`.

## HTTP adapter specifics

- Request DTO names: `<Action><Entity>Request` (e.g. `CreateSubscriptionRequest`).
- Response DTO names: `<Entity>Response` (e.g. `SubscriptionResponse`).
- Request DTOs carry `validate:""` tags and a `.ToInput(orgId string) service.X`
  method (orgId comes from `AuthUserFrom(c)` at the handler).
- Response DTOs carry `json:""` tags only. Mapper: `NewEntityFromEntity(e domain.E) EntityResponse`.
- Nested response DTOs are built inline via the nested mapper:
  `Customer: NewCustomerFromEntity(c)`.
- Handlers NEVER return `domain.X` directly — that would leak the
  (intentionally tag-free) domain type and produce `OrgId` instead of `org_id`
  in the JSON output.

## Workflow adapter specifics (Hatchet / Temporal)

- Workflow step inputs are serialized over the durable log. They are JSON.
- Moving a Go type to a different package does NOT change the JSON shape —
  serialization is field-name-based, not type-name-based.
- Therefore the input-types-to-service-package sweep is safe as long as
  **field names are preserved**.
- For prod-style deploys with in-flight tasks: drain workers → deploy → restart.

## Adding a new entity (checklist)

1. Define the **domain** type in `internal/core/domain/<entity>.go`. Pure Go,
   no tags. ID-only references.
2. Define **input types** for the use cases in `internal/core/service/<entity>_input.go`
   (or inline in the service file if there are 1–2). Plain structs, no tags.
3. Define the **service** in `internal/core/service/<entity>.go`. Takes ports
   in its constructor. Methods accept `service.*Input` types.
4. Define the **port** for its repository in `internal/core/port/`.
5. Implement the **postgres row** at `internal/adapter/postgres/<entity>_row.go`
   with `toDomain` and `<entity>RowFromDomain` mappers.
6. Implement the **repo** at `internal/adapter/postgres/<entity>_repo.go`. Use
   the row type internally; translate at the boundary.
7. Add **HTTP DTOs** in `internal/adapter/http/<entity>_dto.go` (request types)
   and either add to `response.go` or create `<entity>_response.go` (response
   types + `NewEntityFromEntity` mappers).
8. Implement the **HTTP handler** in `internal/adapter/http/<entity>_handler.go`.
   Accept request DTOs via `fuego.ContextWithBody[T]`; return response DTOs.

## Litmus tests

When unsure where something belongs, ask:

- *"Would this type still make sense if there were no use cases (no Create,
  Update, Pause, ...)?"* If yes → `domain/`. If no → `service/`.
- *"Does this type's existence depend on HTTP / GORM / validator?"* If yes,
  it belongs in the relevant adapter.
- *"Could I read this file with no knowledge of the persistence layer and
  still understand the business?"* For files in `core/`, the answer must be
  yes.
````

- [ ] **Step 2: Commit**

```bash
cd /Users/mdwt/dev/gphq/gphq-server
git add docs/internal/hexagonal-mapping-pattern.md
git commit -m "docs: add hexagonal & DDD rules reference

Canonical rulebook for where types live and what tags they may carry.
Domain layer holds aggregates / value objects / domain services / events.
Application layer (internal/core/service/) holds use cases AND their
command/query input types. Adapters own their wire formats (HTTP DTOs,
postgres row types) with explicit mappers at every boundary.

This document is the spec the hexagonal-domain-split plan implements.
"
```

### Task 0.2: Create row_helpers.go (scaffold; populated lazily)

**Files:**
- Create: `internal/adapter/postgres/row_helpers.go`

- [ ] **Step 1: Create the file with a header comment only**

We will add helpers here as later tasks identify them. Starting with an empty package-level placeholder keeps the file present so subsequent tasks can append rather than create.

```go
package postgres

// row_helpers.go houses small utilities shared across <entity>Row → domain
// mappers. Add helpers here as patterns emerge — keep them small and
// purpose-built. Examples that will land later in the plan:
//   - timePtr / ptrTime for nullable timestamps that are not modeled with
//     serializer:nulltime
//   - copyMap[K comparable, V any] for shallow-copying serialized JSON maps
//     when mutation safety matters
```

- [ ] **Step 2: Verify package still builds**

```bash
cd /Users/mdwt/dev/gphq/gphq-server
go build ./internal/adapter/postgres/...
```

Expected: clean exit.

- [ ] **Step 3: Commit**

```bash
git add internal/adapter/postgres/row_helpers.go
git commit -m "scaffold: add internal/adapter/postgres/row_helpers.go

Placeholder for mapper helpers shared across <entity>Row types introduced
by the hexagonal-domain-split refactor.
"
```

### Task 0.3: Move all `*Input` types from `domain/` to `service/`

This is a single sweep done **once, up front**, before any entity is split.
The move is mechanical: every `domain.*Input` type becomes `service.*Input`.
Field names are unchanged (preserves Hatchet/Temporal durable-log JSON
shapes). After this task the domain layer is free of command/input types
forever.

**Files:**
- Create: `internal/core/service/<entity>_input.go` (one per entity that has input types — about 10 files)
- Modify: `internal/core/domain/<entity>.go` (delete the moved type definitions)
- Modify: every Go file that references `domain.*Input` (~119 sites)

#### The 32 types and their target homes

| Input type | Current file | New file |
|---|---|---|
| `CreateSubscriptionInput`, `UpdateSubscriptionInput`, `PauseSubscriptionInput`, `CancelSubscriptionInput`, `ResumeSubscriptionInput`, `UpdateBillingAnchorInput`, `SubscriptionChargeInput` | `domain/subscription*.go` | `service/subscription_input.go` |
| `CreateOrderInput`, `CompleteOrderInput` | `domain/order*.go` | `service/order_input.go` |
| `CompleteCheckoutSessionInput`, `CreateSessionInput` | `domain/session.go` | `service/session_input.go` |
| `CreateCustomerInput` | `domain/customer*.go` | `service/customer_input.go` |
| `CreatePaymentMethodInput`, `UpdatePaymentMethodInput` | `domain/payment_method*.go` | `service/payment_method_input.go` |
| `CreatePriceInput`, `CreateProductPriceInput` | `domain/price.go`, `domain/product_input.go` | `service/price_input.go` |
| `CreateProductInput`, `UpdateProductInput`, `CreateProductVariantInput` | `domain/product_input.go` | `service/product_input.go` |
| `CreateVariantInput`, `UpdateVariantInput` | `domain/variant*.go` | `service/variant_input.go` |
| `CreateWebhookSubscriptionInput` | `domain/webhook_subscription.go` | `service/webhook_subscription_input.go` |
| `CreateDunningCampaignInput`, `PauseDunningCampaignInput`, `ResumeDunningCampaignInput`, `CancelDunningCampaignInput`, `CreateDunningConfigurationInput`, `UpdateDunningConfigurationInput`, `TriggerManualAttemptInput`, `StartDunningWorkflowInput` | `domain/dunning_input.go` | `service/dunning_input.go` |
| `CreatePaymentUpdateTokenInput`, `ActivatePaymentUpdateTokenInput` | `domain/dunning_input.go` | `service/payment_update_token_input.go` |

#### Sweep procedure

- [ ] **Step 1: Confirm the full list**

```bash
cd /Users/mdwt/dev/gphq/gphq-server
grep -rEho 'domain\.[A-Z][a-zA-Z]*Input\b' internal/ --include="*.go" | sort -u
```

Expected: the 32-type list from Background #9. If a new Input has appeared
since this plan was written, add it to the appropriate row above.

- [ ] **Step 2: Create each new `service/<entity>_input.go` file**

For every row in the table above, create the file in `internal/core/service/`
with package `service` and the moved type definitions, **stripping all tags**
(no `json:""` — that's a request-DTO concern now; no `validate:""` —
same reason).

Example: `internal/core/service/subscription_input.go`:

```go
package service

import (
    "getpaidhq/internal/core/domain"
)

// CreateSubscriptionInput is the command DTO for SubscriptionService.Create.
// Use cases (application services) accept this; the HTTP layer maps from its
// CreateSubscriptionRequest via .ToInput(orgId).
type CreateSubscriptionInput struct {
    OrgId              string
    PaymentMethodId    string
    Activate           bool
    Amount             int64
    Currency           string
    BillingInterval    domain.BillingInterval
    BillingIntervalQty int
    Cycles             int
    TrialInterval      domain.BillingInterval
    TrialIntervalQty   int
    Metadata           map[string]string
}

// UpdateSubscriptionInput ...
type UpdateSubscriptionInput struct {
    OrgId    string
    Id       string
    Status   domain.SubscriptionStatus
    Metadata map[string]string
}

// PauseSubscriptionInput ...
type PauseSubscriptionInput struct {
    OrgId  string
    Id     string
    Reason string
}

// ... and so on for the remaining subscription inputs.
```

Repeat for each input file. Copy the field list verbatim from the existing
domain definition; strip the tags.

- [ ] **Step 3: Delete the type definitions from `domain/`**

For each moved type, delete its `type X struct { ... }` block from the source
file in `domain/`. Leave other types in those files untouched.

If a domain file becomes empty (only an unused `package domain` remains),
delete the file. Likely candidates: `domain/dunning_input.go`,
`domain/product_input.go`, `domain/order_input.go` (verify after the sweep).

- [ ] **Step 4: Run a global rewrite of references**

Use ripgrep + sed (or your editor's multi-file find-replace) to rewrite
every reference. The pattern is mechanical:

```bash
# Preview hits per type first:
for t in CreateSubscriptionInput UpdateSubscriptionInput PauseSubscriptionInput \
         CancelSubscriptionInput ResumeSubscriptionInput UpdateBillingAnchorInput \
         SubscriptionChargeInput CreateOrderInput CompleteOrderInput \
         CompleteCheckoutSessionInput CreateSessionInput CreateCustomerInput \
         CreatePaymentMethodInput UpdatePaymentMethodInput CreatePriceInput \
         CreateProductPriceInput CreateProductInput UpdateProductInput \
         CreateProductVariantInput CreateVariantInput UpdateVariantInput \
         CreateWebhookSubscriptionInput CreateDunningCampaignInput \
         PauseDunningCampaignInput ResumeDunningCampaignInput \
         CancelDunningCampaignInput CreateDunningConfigurationInput \
         UpdateDunningConfigurationInput TriggerManualAttemptInput \
         StartDunningWorkflowInput CreatePaymentUpdateTokenInput \
         ActivatePaymentUpdateTokenInput; do
    echo "=== $t ==="
    grep -rln "domain\.$t\b" internal/ --include="*.go"
done
```

Then apply the rewrite (macOS sed):

```bash
for t in CreateSubscriptionInput UpdateSubscriptionInput PauseSubscriptionInput \
         CancelSubscriptionInput ResumeSubscriptionInput UpdateBillingAnchorInput \
         SubscriptionChargeInput CreateOrderInput CompleteOrderInput \
         CompleteCheckoutSessionInput CreateSessionInput CreateCustomerInput \
         CreatePaymentMethodInput UpdatePaymentMethodInput CreatePriceInput \
         CreateProductPriceInput CreateProductInput UpdateProductInput \
         CreateProductVariantInput CreateVariantInput UpdateVariantInput \
         CreateWebhookSubscriptionInput CreateDunningCampaignInput \
         PauseDunningCampaignInput ResumeDunningCampaignInput \
         CancelDunningCampaignInput CreateDunningConfigurationInput \
         UpdateDunningConfigurationInput TriggerManualAttemptInput \
         StartDunningWorkflowInput CreatePaymentUpdateTokenInput \
         ActivatePaymentUpdateTokenInput; do
    grep -rl "domain\.$t\b" internal/ --include="*.go" | while read f; do
        sed -i '' "s/domain\\.$t\\b/service.$t/g" "$f"
    done
done
```

Files referencing `service.X` need a `service` import. Run `goimports`:

```bash
goimports -w ./internal/...
```

- [ ] **Step 5: Build the whole project**

```bash
go build ./...
```

Likely remaining failures and their fixes:

a) **`undefined: domain.NewFromCreateInput`** (or similar constructor):
   `NewFromCreateInput` lives in `domain/subscription.go` and takes a
   `CreateSubscriptionInput` argument. After the move, the function signature
   becomes `func NewFromCreateInput(input service.CreateSubscriptionInput) Subscription`
   — but **`domain/` MUST NOT import `service/`** (that's a layer
   inversion). Move the constructor too:
   
   - Delete `NewFromCreateInput` from `domain/subscription.go`.
   - Add `(input CreateSubscriptionInput) ToSubscription() domain.Subscription`
     as a method on the Input type in `service/subscription_input.go`. Method
     body is the same logic.
   - Replace call sites: `domain.NewFromCreateInput(input)` → `input.ToSubscription()`.

b) **Tests that constructed `domain.CreateXInput{...}` literals**: rewrite as
   `service.CreateXInput{...}`. Tests living in the same package as the
   service (`service_test.go` in `internal/core/service/`) drop the package
   prefix entirely.

c) **Hatchet step shims passing input through as JSON**: the wire shape is
   unchanged. The Go type at the boundary changes from `domain.X` to
   `service.X`. Update the step's input type parameter.

Loop until `go build ./...` is clean.

- [ ] **Step 6: Run all tests**

```bash
go test ./...
go test -tags=integration ./...
```

Expected: PASS. No behavior changed — only package paths.

- [ ] **Step 7: Confirm no `domain.*Input` references remain**

```bash
grep -rn "domain\.\(Create\|Update\|Delete\|Pause\|Cancel\|Resume\|Add\|Remove\|Adjust\|Activate\|Complete\|Trigger\|Start\)[A-Z][a-zA-Z]*Input\b" internal/ --include="*.go"
```

Expected: zero matches.

- [ ] **Step 8: Confirm no `validate:""` tags remain on the moved types**

```bash
grep -n 'validate:' internal/core/service/*_input.go
```

Expected: zero matches. The Input types are tag-free.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "refactor: move *Input types from domain/ to service/

Command/query DTOs are an application-layer concern (use-case shape),
not a domain concern. Per the hexagonal & DDD rules
(docs/internal/hexagonal-mapping-pattern.md):

  Domain  = aggregates + value objects + domain services + events
  Service = use cases + their command/query input types

32 types moved from internal/core/domain/* to internal/core/service/*_input.go,
~119 reference sites rewritten. Input types are passive structs with NO
tags (no json, no validate) — validation lives on HTTP request DTOs that
map to Inputs via .ToInput(orgId).

NewFromCreateInput becomes a method on the Input type (input.ToSubscription())
to preserve the layer rule (domain MUST NOT import service).

Hatchet/Temporal step JSON shapes unchanged (serialization is field-name-
based, not type-name-based; field names are preserved).
"
```

---

## Phase 1 — Execute the Hexagonal Split Across All 20 Entities

No pilot. Execute the split entity-by-entity in **dependency order** (leaves first so their building blocks are in place when composites need them). Each entity goes through the same Playbook (defined once below). Tests stay green between entities — never commit a half-state.

### Execution order (locked)

| Wave | Entities | Why first / why last |
|---|---|---|
| **1. Pure leaves** (no domain relations, no nested in others' read models) | User, Session, Setting, MetadataStore, Org, PspConfig, Refund, ApiKey, WebhookSubscription | Smallest surface; pattern lands cleanly with no relationship complexity. |
| **2. Building blocks** (referenced by others' read models) | Customer, Price, PaymentMethod | Required before anything that nests them. |
| **3. Mid-level** (need building blocks; have own read model) | OrderItem, Variant, Subscription | Each needs at least one wave-2 entity. |
| **4. Composites** (nest multiple of waves 1–3) | Order (needs Customer+OrderItem+Price), Product (needs Variant→Price) | These earn the most-complex read models. |
| **5. Standalone-with-structure** | Payment, Cart | No nested entities but non-trivial JSON-serialized columns. |
| **6. Dunning bundle** (6 entity types in one file) | DunningCampaign, DunningAttempt, DunningCommunication, PaymentUpdateToken, DunningConfiguration, CustomerDunningHistory | Largest single batch; benefits from helpers settled by then. |

### The Playbook (every entity follows this)

For entity `<E>`:

1. **Postgres row + mappers.** Create `internal/adapter/postgres/<entity>_row.go` — `<entity>Row` struct mirroring the domain field list, GORM tags only, `TableName()`, `toDomain()`, `<entity>RowFromDomain()`.
2. **Repo rewrite.** Update `internal/adapter/postgres/<entity>_repo.go` to use the row internally. Translate at the boundary. Add `FindByIds(ctx, orgId, ids []string) ([]domain.E, error)` if the entity is referenced by another entity's read model — this is the batch primitive that prevents N+1 in services.
3. **Refactor domain methods that read embedded relations.** Anywhere a domain method dereferences a struct field (e.g. `s.OrderItem.Price.*`), change the signature to take the related entity explicitly. Update call sites.
4. **Strip GORM + JSON + `TableName()` from domain.** Delete `TableName()`. Remove `gorm:""` and `json:""` tags. Remove embedded relationship struct fields (keep only the FK IDs). Move transient `gorm:"-"` fields out per the Q8 rule (computed fields → methods; create-time-only fields → service result types).
5. **Read model + Query Handler (only when the response nests related entities).** Add `service.<E>Details` struct in `internal/core/service/<entity>_read.go` composed of `domain.E` plus nested domain entities. Add `func (s *<E>Service) GetDetails(ctx, orgId, id) (<E>Details, error)` and `ListDetails(ctx, orgId, pagination) ([]<E>Details, int, error)` that compose via `FindByIds` batch loads.
6. **HTTP request DTOs.** Create `internal/adapter/http/<entity>_request.go` with `<Action><E>Request` types (json + validate tags) and `.ToInput(orgId)` methods returning `service.<Action><E>Input`.
7. **HTTP response DTOs.** Create `internal/adapter/http/<entity>_response.go` with `<E>Response` + `NewEntityFromEntity(domain.E) <E>Response`. For entities with read models, add `New<E>ResponseFromDetails(service.<E>Details) <E>Response` as well. Remove the entity's DTO from `response.go` if present (one-time split).
8. **Handler audit.** Every handler method returns `<E>Response`, never `domain.E`. Every command handler builds the service input via `request.ToInput(orgId)`. Every GET handler with nested response calls the service's `GetDetails` / `ListDetails` method and maps the read model.
9. **Verify.** `go build ./...`, `go test ./...`, `go test -tags=integration ./...`. Fix every failure before commit.
10. **Regenerate openapi.json** (boot the server). Diff — public shape must be unchanged.
11. **Commit.** Message:
    ```
    refactor(<entity>): hexagonal split — domain / postgres row / service read model / DTO

    - domain.<E> is tag-free, references others by ID only
    - <entity>Row owns gorm mapping in postgres adapter
    - service.<E>Details + GetDetails/ListDetails for the composed query (if any)
    - <Action><E>Request → service.<Action><E>Input via ToInput(orgId)
    - <E>Response is the wire shape

    Follows docs/internal/hexagonal-mapping-pattern.md.
    ```

### Reference: Subscription end-to-end (the fully spelled-out example)

The detailed steps that follow show every line of code for the Subscription entity — the most representative case (embedded relations, nulltime columns, JSON metadata, domain methods that read through relations, an existing partial DTO). Use this as the canonical reference; for every other entity, apply the same shape against its actual field list.

**This is not a separate "pilot phase" — it's the worked example. Execute it as part of Wave 3 in its actual position. The other 19 entities follow the same shape, condensed because their field lists are simpler.**

#### Reference example details for Subscription

### Task 1.1: Decouple `Subscription.SetActivationDates` from the embedded OrderItem.Price

`SetActivationDates` reads `s.OrderItem.Price.*`. After the split, `s.OrderItem` doesn't exist on the domain entity. Refactor the method to take `Price` as a parameter. Same for any caller — they already have the price available from the order item.

**Files:**
- Modify: `internal/core/domain/subscription.go` (the `SetActivationDates` method and `NewSubscriptionFromOrderItem`)
- Modify: every call site of `SetActivationDates`

- [ ] **Step 1: Find every call site of SetActivationDates**

```bash
cd /Users/mdwt/dev/gphq/gphq-server
grep -rn "SetActivationDates\|SetActive(" internal/ --include="*.go"
```

Record every location. Expect ≤ 5 call sites total (`SetActivationDates` is called inside `SetActive`).

- [ ] **Step 2: Write a failing unit test for the new signature**

Open `internal/core/domain/subscription_test.go` and add:

```go
func TestSetActivationDates_TakesPriceArgument(t *testing.T) {
    sub := Subscription{
        OrgId:              "org_test",
        Id:                 "sub_test",
        BillingInterval:    BillingIntervalMonth,
        BillingIntervalQty: 1,
    }
    price := Price{
        BillingInterval:    BillingIntervalMonth,
        BillingIntervalQty: 1,
        TrialInterval:      BillingIntervalDay,
        TrialIntervalQty:   7,
        Cycles:             12,
    }

    sub.SetActivationDates(price)

    if sub.TrialEndsAt.IsZero() {
        t.Fatalf("expected TrialEndsAt to be populated from price.TrialInterval, got zero")
    }
    if sub.EndsAt.IsZero() {
        t.Fatalf("expected EndsAt to be populated from price.Cycles, got zero")
    }
    if !sub.RenewsAt.After(sub.StartDate) {
        t.Fatalf("expected RenewsAt to be after StartDate, got %v vs %v", sub.RenewsAt, sub.StartDate)
    }
}
```

- [ ] **Step 3: Run the test to confirm it fails to compile**

```bash
go test ./internal/core/domain/ -run TestSetActivationDates_TakesPriceArgument
```

Expected: compile error — `sub.SetActivationDates(price)` doesn't match current `func (s *Subscription) SetActivationDates() *Subscription`.

- [ ] **Step 4: Refactor `SetActivationDates` to take `Price`**

Edit `internal/core/domain/subscription.go`. Replace the existing `SetActivationDates` method (currently reads `s.OrderItem.Price`) with:

```go
// SetActivationDates initializes the lifecycle date fields (StartDate,
// TrialEndsAt, EndsAt, RenewsAt, CurrentPeriodStart, CurrentPeriodEnd,
// BillingAnchor) from the Price the subscription is created against.
// The Price is passed in explicitly rather than being read off an embedded
// OrderItem — domain entities reference others by ID, not by struct.
func (s *Subscription) SetActivationDates(price Price) *Subscription {
    startDate := time.Now().UTC()
    var trialEndsAt time.Time
    var endsAt time.Time

    if price.TrialInterval != BillingIntervalNone {
        switch price.TrialInterval {
        case "minute":
            trialEndsAt = startDate.Add(time.Minute * time.Duration(price.TrialIntervalQty))
        case "hour":
            trialEndsAt = startDate.Add(time.Hour * time.Duration(price.TrialIntervalQty))
        case "day":
            trialEndsAt = startDate.AddDate(0, 0, price.TrialIntervalQty)
        case "week":
            trialEndsAt = startDate.AddDate(0, 0, price.TrialIntervalQty*7)
        case "month":
            trialEndsAt = startDate.AddDate(0, price.TrialIntervalQty, 0)
        case "year":
            trialEndsAt = startDate.AddDate(price.TrialIntervalQty, 0, 0)
        }
    }

    if price.Cycles > 0 {
        endsAt = calculateNextDate(price.BillingInterval, price.Cycles*price.BillingIntervalQty, startDate)
    }

    s.StartDate = startDate
    s.TrialEndsAt = trialEndsAt
    s.EndsAt = endsAt
    s.RenewsAt = s.CalculateNextBillingDate()
    s.CurrentPeriodStart = startDate
    s.CurrentPeriodEnd = s.RenewsAt
    s.BillingAnchor = startDate.Day()

    return s
}
```

- [ ] **Step 5: Update `SetActive` (calls `SetActivationDates`) to take `Price`**

In the same file, change:

```go
func (s *Subscription) SetActive(payment Payment) *Subscription {
    s.SetActivationDates()
    // ...
```

to:

```go
func (s *Subscription) SetActive(price Price, payment Payment) *Subscription {
    s.SetActivationDates(price)
    s.Status = SubscriptionStatusActive
    if payment.OrgId != "" && payment.Amount > 0 {
        s.LastCharge = payment.CompletedAt
        s.TotalRevenue = payment.Amount
        s.CyclesProcessed++
        renewsAt := s.CalculateNextBillingDate()
        s.RenewsAt = renewsAt
        s.CurrentPeriodStart = s.StartDate
        s.CurrentPeriodEnd = renewsAt
    }
    return s
}
```

- [ ] **Step 6: Fix every call site found in Step 1**

For each location, change `sub.SetActive(payment)` to `sub.SetActive(orderItem.Price, payment)` (the call site already has the order item available — it constructed the subscription from it). For `sub.SetActivationDates()`, change to `sub.SetActivationDates(orderItem.Price)`.

Use `grep` again to confirm zero remaining bare calls:

```bash
grep -rn "SetActivationDates()\|SetActive(payment\|\.SetActive([^)]*) " internal/ --include="*.go"
```

- [ ] **Step 7: Run the test — should pass now**

```bash
go test ./internal/core/domain/ -run TestSetActivationDates_TakesPriceArgument
```

Expected: PASS.

- [ ] **Step 8: Run the full domain suite**

```bash
go test ./internal/core/domain/...
```

Expected: PASS. Pay attention to `subscription_test.go` — existing tests on these methods may have used `Subscription{OrderItem: ...}` literals; fix them too.

- [ ] **Step 9: Run the whole project**

```bash
go build ./...
go test ./...
```

Expected: PASS. Integration tests excluded by default.

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "refactor(domain): SetActivationDates/SetActive take Price explicitly

Removes the implicit dependency on s.OrderItem.Price being loaded on the
Subscription entity. Prep for the hexagonal split — domain entities will
no longer carry embedded related structs.

Test: domain.TestSetActivationDates_TakesPriceArgument
"
```

### Task 1.2: Introduce `subscriptionRow` in postgres

**Files:**
- Create: `internal/adapter/postgres/subscription_row.go`
- Modify (later step): `internal/adapter/postgres/subscription_repo.go`

- [ ] **Step 1: Write the row type with mappers**

Create `internal/adapter/postgres/subscription_row.go`:

```go
package postgres

import (
    "time"

    "getpaidhq/internal/core/domain"
)

// subscriptionRow is the postgres on-the-wire representation of a Subscription.
// Lowercase / internal to this package. GORM tags live here; the domain.Subscription
// type is tag-free.
type subscriptionRow struct {
    OrgId              string                    `gorm:"column:org_id;primaryKey"`
    Id                 string                    `gorm:"column:id;primaryKey"`
    PspId              domain.Gateway            `gorm:"column:psp_id"`
    OrderId            string                    `gorm:"column:order_id"`
    OrderItemId        string                    `gorm:"column:order_item_id"`
    CustomerId         string                    `gorm:"column:customer_id"`
    Status             domain.SubscriptionStatus `gorm:"column:status"`
    PaymentMethodId    string                    `gorm:"column:payment_method_id"`
    StartDate          time.Time                 `gorm:"column:start_date;serializer:nulltime"`
    EndDate            time.Time                 `gorm:"column:end_date;serializer:nulltime"`
    BillingInterval    domain.BillingInterval    `gorm:"column:billing_interval"`
    BillingIntervalQty int                       `gorm:"column:billing_interval_qty"`
    Cycles             int                       `gorm:"column:cycles"`
    BillingAnchor      int                       `gorm:"column:billing_anchor"`
    TrialEndsAt        time.Time                 `gorm:"column:trial_ends_at;serializer:nulltime"`
    CancelAt           time.Time                 `gorm:"column:cancel_at;serializer:nulltime"`
    EndsAt             time.Time                 `gorm:"column:ends_at;serializer:nulltime"`
    LastCharge         time.Time                 `gorm:"column:last_charge;serializer:nulltime"`
    RenewsAt           time.Time                 `gorm:"column:renews_at;serializer:nulltime"`
    CurrentPeriodStart time.Time                 `gorm:"column:current_period_start;serializer:nulltime"`
    CurrentPeriodEnd   time.Time                 `gorm:"column:current_period_end;serializer:nulltime"`
    Retries            int                       `gorm:"column:retries"`
    NextRetryAt        time.Time                 `gorm:"column:next_retry;serializer:nulltime"`
    Currency           string                    `gorm:"column:currency"`
    Amount             int64                     `gorm:"column:amount"`
    Metadata           map[string]string         `gorm:"column:metadata;serializer:json"`
    CyclesProcessed    int                       `gorm:"column:cycles_processed"`
    TotalRevenue       int64                     `gorm:"column:total_revenue"`
    CancelledAt        time.Time                 `gorm:"column:cancelled_at;serializer:nulltime"`
    CreatedAt          time.Time                 `gorm:"column:created_at"`
    UpdatedAt          time.Time                 `gorm:"column:updated_at"`
}

func (subscriptionRow) TableName() string { return "subscriptions" }

// toDomain maps a row to its domain Subscription. No relationships are
// hydrated — those live on a separate result struct (subscriptionWithCustomerRow
// below) when callers need them.
func (r subscriptionRow) toDomain() domain.Subscription {
    return domain.Subscription{
        OrgId:              r.OrgId,
        Id:                 r.Id,
        PspId:              r.PspId,
        OrderId:            r.OrderId,
        OrderItemId:        r.OrderItemId,
        CustomerId:         r.CustomerId,
        Status:             r.Status,
        PaymentMethodId:    r.PaymentMethodId,
        StartDate:          r.StartDate,
        EndDate:            r.EndDate,
        BillingInterval:    r.BillingInterval,
        BillingIntervalQty: r.BillingIntervalQty,
        Cycles:             r.Cycles,
        BillingAnchor:      r.BillingAnchor,
        TrialEndsAt:        r.TrialEndsAt,
        CancelAt:           r.CancelAt,
        EndsAt:             r.EndsAt,
        LastCharge:         r.LastCharge,
        RenewsAt:           r.RenewsAt,
        CurrentPeriodStart: r.CurrentPeriodStart,
        CurrentPeriodEnd:   r.CurrentPeriodEnd,
        Retries:            r.Retries,
        NextRetryAt:        r.NextRetryAt,
        Currency:           r.Currency,
        Amount:             r.Amount,
        Metadata:           r.Metadata,
        CyclesProcessed:    r.CyclesProcessed,
        TotalRevenue:       r.TotalRevenue,
        CancelledAt:        r.CancelledAt,
        CreatedAt:          r.CreatedAt,
        UpdatedAt:          r.UpdatedAt,
    }
}

// subscriptionRowFromDomain produces a row for INSERT/UPDATE.
func subscriptionRowFromDomain(e domain.Subscription) subscriptionRow {
    return subscriptionRow{
        OrgId:              e.OrgId,
        Id:                 e.Id,
        PspId:              e.PspId,
        OrderId:            e.OrderId,
        OrderItemId:        e.OrderItemId,
        CustomerId:         e.CustomerId,
        Status:             e.Status,
        PaymentMethodId:    e.PaymentMethodId,
        StartDate:          e.StartDate,
        EndDate:            e.EndDate,
        BillingInterval:    e.BillingInterval,
        BillingIntervalQty: e.BillingIntervalQty,
        Cycles:             e.Cycles,
        BillingAnchor:      e.BillingAnchor,
        TrialEndsAt:        e.TrialEndsAt,
        CancelAt:           e.CancelAt,
        EndsAt:             e.EndsAt,
        LastCharge:         e.LastCharge,
        RenewsAt:           e.RenewsAt,
        CurrentPeriodStart: e.CurrentPeriodStart,
        CurrentPeriodEnd:   e.CurrentPeriodEnd,
        Retries:            e.Retries,
        NextRetryAt:        e.NextRetryAt,
        Currency:           e.Currency,
        Amount:             e.Amount,
        Metadata:           e.Metadata,
        CyclesProcessed:    e.CyclesProcessed,
        TotalRevenue:       e.TotalRevenue,
        CancelledAt:        e.CancelledAt,
        CreatedAt:          e.CreatedAt,
        UpdatedAt:          e.UpdatedAt,
    }
}
```

- [ ] **Step 2: Verify the file compiles**

```bash
cd /Users/mdwt/dev/gphq/gphq-server
go build ./internal/adapter/postgres/...
```

Expected: clean.

### Task 1.3: Update subscription_repo.go to use rows

`subscription_repo.go` currently `Preload("Customer")` on three methods. The composite query stays — we add a parallel `subscriptionWithCustomerRow` that GORM can Preload into, and the repo decides per-method whether to project to domain.Subscription alone or to a `(Subscription, Customer)` pair.

**Files:**
- Modify: `internal/adapter/postgres/subscription_row.go` (add composite row)
- Modify: `internal/adapter/postgres/subscription_repo.go`
- Modify: `internal/core/port/repository.go` (likely no change — repo still returns `domain.Subscription`)

- [ ] **Step 1: Add composite row + helper in subscription_row.go**

Append to `internal/adapter/postgres/subscription_row.go`:

```go
// customerRow lives in customer_row.go (added by Task 2.x).  This struct uses
// it via gorm's foreignKey relationship for Preload-based queries.  The
// composite is internal to this package; service code receives a domain
// pair, not the row pair.
type subscriptionWithCustomerRow struct {
    subscriptionRow
    Customer customerRow `gorm:"foreignKey:CustomerId,OrgId;references:Id,OrgId"`
}

// toDomainPair returns the subscription as a domain entity plus its preloaded
// customer (also as a domain entity).  Used by repo methods that previously
// did Preload("Customer").
func (r subscriptionWithCustomerRow) toDomainPair() (domain.Subscription, domain.Customer) {
    return r.subscriptionRow.toDomain(), r.Customer.toDomain()
}
```

NOTE: `customerRow` does not exist yet. We will create it in Task 2.1 (Customer comes first in Phase 2 because it is a precondition for Subscription's relationship preload). The pilot's repo update temporarily uses `Preload("Customer")` against the existing `domain.Customer` until Task 2.1 lands; on a clean repo, **defer this composite addition until after Customer's row is in place**.

To keep the pilot self-contained, we instead change the repo to **not Preload internally**; service callers that need the customer call `customerRepo.FindById(ctx, orgId, sub.CustomerId)`.

Replace the append above with:

```go
// (no composite row in this task. The repo's three Preload sites are
// converted to bare queries; downstream service code does a separate
// CustomerRepo.FindById when it needs the customer. See PR description for
// the audit of those sites.)
```

- [ ] **Step 2: Audit service code for `sub.Customer.*` reads**

```bash
cd /Users/mdwt/dev/gphq/gphq-server
grep -rn "\.Customer\.\|sub\.Customer\b\|subscription\.Customer\b" internal/core/service/ internal/adapter/hatchet/ internal/adapter/temporal/ --include="*.go" | grep -v _test
```

Record every site. For each, change the site to either:
- (a) call `customerRepo.FindById(ctx, orgId, sub.CustomerId)` and read from that local, or
- (b) accept the customer as a parameter alongside the subscription.

This is mechanical. The Customer is read for: email/name in dunning comms, email in payment-update tokens, email in webhook payloads, and similar. Each is an obvious `+ var, err := r.customerRepo.FindById(...)` insertion.

- [ ] **Step 3: Rewrite subscription_repo.go to use rows**

Replace `internal/adapter/postgres/subscription_repo.go` with (each method follows the row pattern):

```go
package postgres

import (
    "context"
    "time"

    "gorm.io/gorm"
    "gorm.io/gorm/clause"

    "getpaidhq/internal/core/domain"
    "getpaidhq/internal/core/port"
)

type SubscriptionRepo struct {
    db *gorm.DB
}

func NewSubscriptionRepo(db *gorm.DB) port.SubscriptionRepository {
    return &SubscriptionRepo{db: db}
}

func (r *SubscriptionRepo) FindById(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
    var row subscriptionRow
    err := dbFromCtx(ctx, r.db).
        Scopes(OrgScope(orgId)).
        Where("id = ?", id).
        First(&row).Error
    if err != nil {
        return domain.Subscription{}, translateErr(err)
    }
    return row.toDomain(), nil
}

// FindByIdForUpdate row-locking variant; MUST be in a tx.
func (r *SubscriptionRepo) FindByIdForUpdate(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
    var row subscriptionRow
    err := dbFromCtx(ctx, r.db).
        Clauses(clause.Locking{Strength: "UPDATE"}).
        Scopes(OrgScope(orgId)).
        Where("id = ?", id).
        First(&row).Error
    if err != nil {
        return domain.Subscription{}, translateErr(err)
    }
    return row.toDomain(), nil
}

func (r *SubscriptionRepo) Create(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
    entity.Metadata = emptyIfNil(entity.Metadata)
    row := subscriptionRowFromDomain(entity)
    if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
        return domain.Subscription{}, err
    }
    return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) Update(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
    row := subscriptionRowFromDomain(entity)
    if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
        return domain.Subscription{}, err
    }
    return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
    var rows []subscriptionRow
    err := dbFromCtx(ctx, r.db).
        Scopes(OrgScope(orgId)).
        Where("order_id = ?", orderId).
        Find(&rows).Error
    if err != nil {
        return nil, err
    }
    out := make([]domain.Subscription, len(rows))
    for i, row := range rows {
        out[i] = row.toDomain()
    }
    return out, nil
}

func (r *SubscriptionRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Subscription, int, error) {
    var rows []subscriptionRow
    var count int64
    err := dbFromCtx(ctx, r.db).Model(&subscriptionRow{}).
        Scopes(OrgScope(orgId)).
        Count(&count).Error
    if err != nil {
        return nil, 0, err
    }
    err = dbFromCtx(ctx, r.db).
        Scopes(OrgScope(orgId), Paginate(p)).
        Find(&rows).Error
    if err != nil {
        return nil, 0, err
    }
    out := make([]domain.Subscription, len(rows))
    for i, row := range rows {
        out[i] = row.toDomain()
    }
    return out, int(count), nil
}

// FindDueForBilling — see the comment on domain.Subscription.IsDueForBilling.
func (r *SubscriptionRepo) FindDueForBilling(ctx context.Context, orgId string, now time.Time) ([]domain.Subscription, error) {
    var rows []subscriptionRow
    err := dbFromCtx(ctx, r.db).
        Scopes(OrgScope(orgId)).
        Where(
            r.db.Where("status = ? AND renews_at <= ?", domain.SubscriptionStatusActive, now).
                Or("status = ? AND next_retry <= ?", domain.SubscriptionStatusPastDue, now).
                Or("status = ? AND trial_ends_at <= ?", domain.SubscriptionStatusTrial, now),
        ).
        Find(&rows).Error
    if err != nil {
        return nil, err
    }
    out := make([]domain.Subscription, len(rows))
    for i, row := range rows {
        out[i] = row.toDomain()
    }
    return out, nil
}

func (r *SubscriptionRepo) FindUpcomingRenewals(ctx context.Context, orgId string, now time.Time, within time.Duration) ([]domain.Subscription, error) {
    var rows []subscriptionRow
    err := dbFromCtx(ctx, r.db).
        Scopes(OrgScope(orgId)).
        Where("status = ? AND renews_at > ? AND renews_at <= ?",
            domain.SubscriptionStatusActive, now, now.Add(within)).
        Find(&rows).Error
    if err != nil {
        return nil, err
    }
    out := make([]domain.Subscription, len(rows))
    for i, row := range rows {
        out[i] = row.toDomain()
    }
    return out, nil
}

// (any other methods present in the existing file get the same row treatment;
// preserve their query semantics exactly. Read the existing repo file in
// full before deleting anything.)
```

When applying this, **read the actual existing file first** and replicate every method one-for-one, applying the row-translation pattern. Do not drop methods that this plan didn't list — the listing here is illustrative of the pattern, not exhaustive.

- [ ] **Step 4: Build the postgres package alone**

```bash
go build ./internal/adapter/postgres/...
```

Expected: clean. If the build fails because something else (e.g. `dunning_repo.go`) references the `domain.Subscription` field `Customer`, **stop and audit those sites first** — they will be cleaned up in Phase 2 but the build needs to be green between commits.

If anything is broken outside the repos we're touching (e.g. service that did `sub.Customer.Email`), apply Step 2's separate-fetch fix at those sites.

- [ ] **Step 5: Run the subscription tests**

```bash
go test ./internal/adapter/postgres/ -run TestSubscription
go test -tags=integration ./internal/adapter/postgres/ -run TestSubscription
```

Expected: PASS. The integration tests use `testDB(t)` — confirm the DB still works against the row schema (column names are unchanged, so it must).

- [ ] **Step 6: Run the full project build & test**

```bash
go build ./...
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/adapter/postgres/subscription_row.go internal/adapter/postgres/subscription_repo.go
# plus any service files touched in Step 2
git add internal/core/service/ internal/adapter/hatchet/ internal/adapter/temporal/
git commit -m "refactor(postgres): introduce subscriptionRow + mappers; repo uses rows

Pilot of the hexagonal-domain-split pattern on Subscription. GORM tags
no longer live on the domain entity — they move to subscriptionRow in
the postgres adapter. Repos translate at the boundary via toDomain /
subscriptionRowFromDomain.

The three subscription Preload(\"Customer\") sites are replaced with
explicit customerRepo.FindById calls at the service-layer consumers
(see services audited).

Follows docs/internal/hexagonal-mapping-pattern.md.
"
```

### Task 1.4: Strip GORM and JSON tags from domain.Subscription

**Files:**
- Modify: `internal/core/domain/subscription.go`

- [ ] **Step 1: Replace the `Subscription` struct definition with tag-free fields**

Edit `internal/core/domain/subscription.go`. Find the `type Subscription struct { ... }` block and replace it with:

```go
type Subscription struct {
    OrgId           string
    Id              string
    PspId           Gateway
    OrderId         string
    OrderItemId     string
    CustomerId      string
    Status          SubscriptionStatus
    PaymentMethodId string

    StartDate          time.Time
    EndDate            time.Time
    BillingInterval    BillingInterval
    BillingIntervalQty int
    Cycles             int
    BillingAnchor      int

    TrialEndsAt time.Time
    CancelAt    time.Time
    EndsAt      time.Time
    LastCharge  time.Time
    RenewsAt    time.Time

    CurrentPeriodStart time.Time
    CurrentPeriodEnd   time.Time

    Retries     int
    NextRetryAt time.Time

    Currency        string
    Amount          int64
    Metadata        map[string]string
    CyclesProcessed int
    TotalRevenue    int64
    CancelledAt     time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

Note: `Customer Customer` and `OrderItem OrderItem` embedded fields are **removed**. The domain holds only IDs.

- [ ] **Step 2: Remove the `TableName` method**

Find and delete:

```go
func (Subscription) TableName() string { return "subscriptions" }
```

This method lives on `subscriptionRow` now.

- [ ] **Step 3: Remove the `OrderItem` embedding from `NewSubscriptionFromOrderItem`**

The constructor previously set `OrderItem: item` on the returned Subscription. Drop that line. (The order item's Price was used by `SetActivationDates` — that's now passed in by the caller, so the constructor no longer needs to embed.)

```go
func NewSubscriptionFromOrderItem(item OrderItem) Subscription {
    return Subscription{
        OrgId:              item.OrgId,
        Id:                 lib.GenerateId("sub"),
        OrderId:            item.OrderId,
        OrderItemId:        item.Id,
        Status:             SubscriptionStatusPending,
        BillingInterval:    item.Price.BillingInterval,
        BillingIntervalQty: item.Price.BillingIntervalQty,
        Cycles:             item.Price.Cycles,
        Retries:            0,
        Currency:           string(item.Price.Currency),
        Amount:             item.Price.UnitPrice,
        CyclesProcessed:    0,
        TotalRevenue:       0,
        CreatedAt:          time.Now().UTC(),
        UpdatedAt:          time.Now().UTC(),
    }
}
```

- [ ] **Step 4: Build the domain package alone**

```bash
go build ./internal/core/domain/...
```

Expected: clean.

- [ ] **Step 5: Build the whole project — fix every site that referenced sub.Customer or sub.OrderItem**

```bash
go build ./...
```

This will fail wherever code reads `sub.Customer.X` or `sub.OrderItem.X`. For every failure:

- Reads of `sub.Customer.*`: introduce a `customer, err := customerRepo.FindById(ctx, sub.OrgId, sub.CustomerId)` immediately before and rewrite the read to `customer.*`.
- Reads of `sub.OrderItem.*` (likely the Price): fetch the order item explicitly via the order item repo, and the Price via the price repo, in the same way. Or, where the service constructed the Subscription from a known OrderItem (e.g. `subscriptionService.CreateFromOrderItem`), pass the OrderItem along the call chain to wherever it's needed.

Loop until `go build ./...` is clean. Do not commit a half-state.

- [ ] **Step 6: Run domain tests; fix any that constructed Subscription with embedded Customer/OrderItem**

```bash
go test ./internal/core/domain/...
```

Existing tests use literals like `Subscription{Customer: Customer{...}, OrderItem: OrderItem{...}}`. Change those to pass the related entity to the method directly (e.g. `sub.SetActive(orderItem.Price, payment)`). The shape of the test stays — only the wiring changes.

- [ ] **Step 7: Run integration tests for subscription**

```bash
go test -tags=integration ./internal/adapter/postgres/ -run TestSubscription
go test -tags=integration ./internal/adapter/postgres/ -run TestBilling
```

Expected: PASS. If GORM auto-migration in the integration test path references the old `Subscription` struct, also drop those references — auto-migration must use the row types (see Task 1.5).

- [ ] **Step 8: Run all tests**

```bash
go build ./...
go test ./...
go test -tags=integration ./...
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/core/domain/subscription.go
git add -A
git commit -m "refactor(domain): Subscription is tag-free, ID-only references

Strips gorm/json tags and the embedded Customer/OrderItem relationships
from domain.Subscription. The DB-row representation lives in postgres
as subscriptionRow (introduced in the previous commit). The public API
shape continues to be served by SubscriptionResponse, which was already
nesting CustomerResponse via NewCustomerFromEntity.

Service code that previously read sub.Customer.X / sub.OrderItem.X now
fetches the related entity through its repo at the call site.
"
```

### Task 1.5: Verify GORM auto-migration uses the row types

`docker compose` / dev startup runs schema sync against Prisma, not GORM auto-migrate, so this is mostly a no-op — but the testcontainer-based integration suite does call `db.AutoMigrate(...)` somewhere in `setup_test.go` or a sibling. Confirm it lists the **row types**, not the domain types.

**Files:**
- Read: `internal/adapter/postgres/setup_test.go`
- Maybe modify: same file

- [ ] **Step 1: Inspect setup_test.go for AutoMigrate**

```bash
grep -n "AutoMigrate" internal/adapter/postgres/setup_test.go internal/adapter/postgres/database.go
```

- [ ] **Step 2: If AutoMigrate lists `&domain.Subscription{}`, change to `&subscriptionRow{}`**

(Other entity references stay as `&domain.X{}` for now — we'll update them as each Phase 2 task lands.)

- [ ] **Step 3: Re-run integration tests**

```bash
go test -tags=integration ./internal/adapter/postgres/...
```

Expected: PASS.

- [ ] **Step 4: Commit if changes were needed**

```bash
git add -A
git commit -m "test(postgres): integration setup AutoMigrates subscriptionRow

The testcontainer schema sync had been pointing at domain.Subscription;
now points at the row type, which is where the gorm tags live.
"
```

### Task 1.6: Update SubscriptionResponse + handler audit

The DTO `SubscriptionResponse` already nests a `CustomerResponse`. Since `domain.Subscription` no longer carries `Customer`, the mapper `NewSubscriptionFromEntity(entity domain.Subscription)` cannot fill it. Two options — pick A:

A. Change the mapper signature to `NewSubscriptionFromEntity(sub domain.Subscription, customer domain.Customer) SubscriptionResponse` and require the handler to fetch the Customer before mapping. The handler always has org access; the additional fetch is one repo call.

B. Drop `Customer` from `SubscriptionResponse`. **This is an API break** — `gphq-web` reads `subscription.customer.email`. **Rejected.**

**Files:**
- Modify: `internal/adapter/http/response.go`
- Modify: `internal/adapter/http/subscription_handler.go`

- [ ] **Step 1: Change `NewSubscriptionFromEntity` signature**

In `internal/adapter/http/response.go`:

```go
// Before:
// func NewSubscriptionFromEntity(entity domain.Subscription) SubscriptionResponse

// After:
func NewSubscriptionFromEntity(sub domain.Subscription, customer domain.Customer) SubscriptionResponse {
    return SubscriptionResponse{
        Id:                 sub.Id,
        OrderId:            sub.OrderId,
        OrderItemId:        sub.OrderItemId,
        Customer:           NewCustomerFromEntity(customer),
        Status:             sub.Status,
        PaymentMethodId:    sub.PaymentMethodId,
        StartDate:          sub.StartDate,
        EndDate:            sub.EndDate,
        BillingInterval:    sub.BillingInterval,
        BillingIntervalQty: sub.BillingIntervalQty,
        Cycles:             sub.Cycles,
        BillingAnchor:      sub.BillingAnchor,
        TrialEndsAt:        sub.TrialEndsAt,
        CancelAt:           sub.CancelAt,
        EndsAt:             sub.EndsAt,
        LastCharge:         sub.LastCharge,
        RenewsAt:           sub.RenewsAt,
        CurrentPeriodStart: sub.CurrentPeriodStart,
        CurrentPeriodEnd:   sub.CurrentPeriodEnd,
        Retries:            sub.Retries,
        NextRetryAt:        sub.NextRetryAt,
        Currency:           sub.Currency,
        Amount:             sub.Amount,
        Metadata:           sub.Metadata,
        CyclesProcessed:    sub.CyclesProcessed,
        TotalRevenue:       sub.TotalRevenue,
        CancelledAt:        sub.CancelledAt,
        CreatedAt:          sub.CreatedAt,
        UpdatedAt:          sub.UpdatedAt,
    }
}
```

- [ ] **Step 2: Update every handler that calls it**

In `internal/adapter/http/subscription_handler.go`, every method that maps a subscription to a response must first fetch the customer. Inject a `CustomerService` (or repo) into the handler if not already present; the org handler already has access.

Pattern:

```go
func (s *SubscriptionHandler) Get(c fuego.ContextNoBody) (SubscriptionResponse, error) {
    authUser := AuthUserFrom(c)
    sub, err := s.subsService.FindById(c.Context(), authUser.OrgId, c.PathParam("id"))
    if err != nil {
        return SubscriptionResponse{}, NewApiErrorFromError(err)
    }
    customer, err := s.customerService.FindById(c.Context(), authUser.OrgId, sub.CustomerId)
    if err != nil {
        return SubscriptionResponse{}, NewApiErrorFromError(err)
    }
    return NewSubscriptionFromEntity(sub, customer), nil
}
```

Apply the same change to `Update`, `Pause`, `Cancel`, `Resume`, `UpdateBillingAnchor`, and the List method (for List, fetch customers in a single batched query if the customer repo exposes one; otherwise fetch per-row and document the N+1 with a TODO referring to the customer repo task to add `FindByIds`).

The handler may need a new dependency. Update its constructor and `app.go` wiring.

- [ ] **Step 3: Update handler tests that expected the old signature**

```bash
go test ./internal/adapter/http/ -run TestSubscription
```

Adjust any test setup to pre-create a customer if it didn't already; the existing test fixtures likely include one (via the order creation flow).

- [ ] **Step 4: Run full test pass**

```bash
go build ./...
go test ./...
go test -tags=integration ./...
```

Expected: PASS.

- [ ] **Step 5: Regenerate openapi.json**

```bash
go run . & sleep 5 && kill %1 2>/dev/null || true
# (the server's Fuego config writes openapi.json to repo root on Run())
```

Or, if there's a `go run ./cmd/openapi-export` (the workspace CLAUDE.md mentions this), use it instead. Check `git diff openapi.json` to confirm the SubscriptionResponse schema is unchanged from the client perspective.

```bash
git diff openapi.json | head -100
```

Expected: no meaningful difference for `SubscriptionResponse` schema (the public shape is preserved).

- [ ] **Step 6: Commit**

```bash
git add internal/adapter/http/response.go internal/adapter/http/subscription_handler.go internal/config/app.go openapi.json
git commit -m "refactor(http): NewSubscriptionFromEntity takes (sub, customer) explicitly

Now that domain.Subscription no longer embeds Customer, the response
mapper requires the customer to be passed in. Handlers fetch the
customer alongside the subscription before mapping. The API contract
is unchanged — SubscriptionResponse still carries the nested customer.
"
```

### Task 1.7: Convert remaining subscription handler returns to DTO

Currently `Update`, `Pause`, etc. return `domain.Subscription` (which is now tag-free Go — it would serialize with default Go field names like `OrgId`, not `org_id`, **breaking the API**). Every one of those must return `SubscriptionResponse`.

**Files:**
- Modify: `internal/adapter/http/subscription_handler.go`

- [ ] **Step 1: Audit every return type in subscription_handler.go**

```bash
grep -n "domain.Subscription\b" internal/adapter/http/subscription_handler.go
```

- [ ] **Step 2: Change every return type to SubscriptionResponse**

For each method, replace:

```go
func (s *SubscriptionHandler) Pause(c fuego.ContextWithBody[PauseSubscriptionRequest]) (domain.Subscription, error) {
    // ...
    return subscription, nil
}
```

with:

```go
func (s *SubscriptionHandler) Pause(c fuego.ContextWithBody[PauseSubscriptionRequest]) (SubscriptionResponse, error) {
    // ...
    customer, err := s.customerService.FindById(c.Context(), authUser.OrgId, subscription.CustomerId)
    if err != nil {
        return SubscriptionResponse{}, NewApiErrorFromError(err)
    }
    return NewSubscriptionFromEntity(subscription, customer), nil
}
```

- [ ] **Step 3: Run all subscription handler tests**

```bash
go test ./internal/adapter/http/ -run TestSubscription
```

Expected: PASS.

- [ ] **Step 4: Regenerate + diff openapi.json**

```bash
# regenerate via server boot; diff to confirm only metadata changed
git diff openapi.json
```

If `domain.Subscription` was previously exposed as a separate schema in OpenAPI (because handler return types referenced it), it'll disappear. That's expected and desired — only the DTO should be reachable from the spec.

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/http/subscription_handler.go openapi.json
git commit -m "refactor(http): subscription handler returns SubscriptionResponse uniformly

Previously Update/Pause/Cancel/Resume/UpdateBillingAnchor returned
domain.Subscription directly, leaking the (now tag-free) domain type
into the API response. They now map through NewSubscriptionFromEntity
like Get and List.
"
```

### Task 1.8: Pilot review checkpoint

Before fanning out to the remaining 19 entities, sanity-check the pattern.

- [ ] **Step 1: Confirm no GORM in domain.Subscription**

```bash
grep -n "gorm:" internal/core/domain/subscription.go
```

Expected: zero matches.

- [ ] **Step 2: Confirm no JSON tag in domain.Subscription**

```bash
grep -n 'json:"' internal/core/domain/subscription.go
```

Expected: zero matches on the persisted entity. Input types like `CreateSubscriptionInput` MAY still carry json tags if they're shared between service and HTTP — assess separately when each input type is touched in Phase 2. The cleanest end state moves all `*Input` json tags into matching `*Request` DTOs.

- [ ] **Step 3: Confirm row file builds and the repo uses it**

```bash
grep -n "subscriptionRow" internal/adapter/postgres/subscription_repo.go
```

Expected: row type referenced; `domain.Subscription` only used at the boundary (return values).

- [ ] **Step 4: All tests still pass**

```bash
go test ./... && go test -tags=integration ./...
```

- [ ] **Step 5: Self-review the pattern**

Read the three changed files end-to-end: `internal/core/domain/subscription.go`, `internal/adapter/postgres/subscription_row.go`, `internal/adapter/postgres/subscription_repo.go`. Ask:
- Does any abstraction look unnecessary? (Consolidate before repeating 19 times.)
- Is the boilerplate big enough that a small helper would pay for itself? (If yes, add it to `row_helpers.go` now.)
- Does the row's `toDomain` need a `toDomainMany([]row)` helper? If 4+ repo methods all do the for-loop, yes — add it to `row_helpers.go` as a generic.

Tentative helper (add if confirmed useful):

```go
// In row_helpers.go
func mapRows[R any, D any](rows []R, fn func(R) D) []D {
    out := make([]D, len(rows))
    for i, r := range rows {
        out[i] = fn(r)
    }
    return out
}
```

If added, refactor subscription_repo.go to use it (`rows.toDomainSlice()` or `mapRows(rows, subscriptionRow.toDomain)`).

- [ ] **Step 6: Pause / hand back to user**

This is the checkpoint described in the executing-plans skill. The plan continues in Phase 2 with the next entity, but a human should look at the pattern in code before authorizing fan-out.

---

## Phase 2 — DEPRECATED — superseded by Phase 1 above

The original plan separated a Subscription "pilot" from the rest of the entities. The user revised this: no pilot, execute the full sweep in dependency order. Phase 1 above now covers all 20 entities; Phase 2's playbook and inventory were merged into Phase 1.

The text below is retained only as historical reference of the per-entity quirks table — the inventory of which entities have which serializer columns, which have embedded relationships, which already have response DTOs in `response.go`. **Execute against Phase 1's Playbook, not this section.**

<details>
<summary>Historical: per-entity quirks (now merged into Phase 1)</summary>

Assumes Task 0.3 has already moved all `*Input` types to `internal/core/service/`. For entity `<E>` with table `<table>`:

1. **Create `internal/adapter/postgres/<entity>_row.go`** with `<entity>Row` struct, `TableName()`, `toDomain()`, `<entity>RowFromDomain()`. Mirror the field list from `internal/core/domain/<entity>.go`. Preserve `serializer:nulltime` / `serializer:json` exactly.
2. **Update `internal/adapter/postgres/<entity>_repo.go`** to use the row internally. Translate at the boundary. Use `mapRows` helper if introduced in Task 1.8.
3. **Add `internal/adapter/postgres/<entity>_row.go` to AutoMigrate** in `setup_test.go` (or wherever the integration suite lists tables) — if AutoMigrate is in fact used (Task 1.5 verified this for Subscription; verify the same way per entity).
4. **Strip `gorm:""` and `json:""` tags + `TableName()` from `internal/core/domain/<entity>.go`.** Remove any embedded relationship struct fields (keep only the FK ids). Where domain methods read through a relation, refactor them to take the related entity as a parameter (see Task 1.1 for the pattern).
5. **Fix every compile error from step 4** by either fetching the related entity at the call site or threading it through the chain.
6. **Create the HTTP DTOs** at `internal/adapter/http/<entity>_dto.go`. This file holds:
   - `<Action><Entity>Request` types with `json:""` + `validate:""` tags.
   - A `.ToInput(orgId string) service.<Action><Entity>Input` method on each request (injects orgId from auth at the handler).
   - Where a response DTO doesn't already exist in `response.go`, add `<Entity>Response` + `NewEntityFromEntity(domain.E) <Entity>Response` here. (Existing response DTOs in `response.go` stay there — moving them is out of scope.)
7. **Audit every handler method** that:
   - Returns `domain.<E>` → switch to `<E>Response`.
   - Accepts `domain.<Action><E>Input` literally → switch to building the input via `request.ToInput(orgId)`.
8. **Run** `go build ./...`, `go test ./...`, `go test -tags=integration ./internal/adapter/postgres/...`. Fix every failure before commit.
9. **Regenerate** `openapi.json` (boot the server). Diff. Confirm the public shape is preserved (or the change is intentional and documented).
10. **Commit** with a message of the form:
    ```
    refactor(<entity>): hexagonal split — domain / postgres row / DTO

    - domain.<E> is tag-free
    - <entity>Row in postgres holds the gorm mapping
    - <Action><E>Request DTOs carry json+validate; .ToInput(orgId) maps
      to service.<Action><E>Input
    - <E>Response is the wire shape; NewEntityFromEntity at the boundary

    Follows docs/internal/hexagonal-mapping-pattern.md.
    ```

### Order of attack

Sequenced by dependency. Customer is first because Subscription's audit (Phase 1) leaves customer-repo calls in service code that will be cleaner once Customer is fully split too. Order/OrderItem precede Product/Variant/Price because the latter are referenced through the former. Dunning is last because it's the biggest chunk (6 entities) and benefits from the helper helpers being settled.

| # | Entity | Group | Special notes |
|---:|---|---|---|
| 2.1 | Customer + CustomerCohort + Cohort | A | Customer has DTO already; CustomerCohort/Cohort don't (no handlers today — confirm with `grep -rn CustomerCohort internal/adapter/http`). If unused by HTTP, skip DTO step for those two. |
| 2.2 | Order | C | Embedded `Customer` and `Items`. Repo introduces `orderWithCustomerAndItemsRow` composite (or two Preloads → composite). DTO `OrderResponse` continues to nest customer + items. |
| 2.3 | OrderItem | C | Embedded `Price`. Composite row needed. DTO `OrderItemResponse` continues to nest price. |
| 2.4 | Product | C | Embedded `Variants []Variant`. Composite row with `Preload("Variants")`. |
| 2.5 | Variant | C | Embedded `Prices []Price`. Composite row with `Preload("Prices")`. |
| 2.6 | Price | B | No relations. `Metadata` JSON. |
| 2.7 | Payment | B | `Metadata` JSON. No relations. Already has DTO. |
| 2.8 | PaymentMethod | B | `BillingAddress` (Address value object) JSON, `Details any` JSON, `Metadata` JSON. **No DTO today — create `payment_method_dto.go`.** |
| 2.9 | Refund | A | Trivial. **No DTO today.** |
| 2.10 | PspConfig | A | Has DTO (`GatewayResponse`). Confirm naming is intentional — `domain.PspConfig` ↔ `GatewayResponse` is jarring but documented. Decide whether to rename `GatewayResponse` → `PspConfigResponse` (preferred) — note that this is an API rename and requires updating SDK/web consumers. Default: **defer rename to a follow-up; keep `GatewayResponse` for this refactor.** |
| 2.11 | Org | A | `Metadata` JSON. **No DTO today.** Org endpoints are sensitive (auth/onboarding); audit carefully. |
| 2.12 | ApiKey | A | `RawKey gorm:"-"` field — preserve "never persisted, returned exactly once at creation" semantics. **No DTO today.** The response DTO must NOT include `KeyHash`; the request creation flow returns the raw key. |
| 2.13 | Setting | A | `Value` is a string with `serializer:json` (oddly tagged but it's `string`, so the serializer round-trips a JSON-quoted string — preserve exactly). **No DTO today.** |
| 2.14 | MetadataStore | A | Triple primary key. No JSON serializer. **No DTO if no HTTP handler — verify.** |
| 2.15 | Session | A | Small. **No DTO today**, but `CreateSessionResponse` exists in the domain file — move it to HTTP DTOs. |
| 2.16 | WebhookSubscription | A | `Events []string` JSON. **No DTO today.** Note `OrgID` (not `OrgId`) — preserve casing. |
| 2.17 | User | A | Tiny. `Password gorm:"-"`. **No DTO today.** Confirm handlers exist (`user_handler.go`). |
| 2.18 | Cart | B | `Data CartData` JSON, `Metadata` JSON, plus `Status` and `Total` `gorm:"-"` fields (derived). Has DTO (`CartResponse`). |
| 2.19 | Dunning bundle (DunningCampaign, DunningAttempt, DunningCommunication, PaymentUpdateToken, DunningConfiguration, CustomerDunningHistory) | D | Largest single change. 6 entities in one domain file → 6 rows in postgres. **No response DTOs exist today** (only request DTOs in `dunning_dto.go`). Create `dunning_response_dto.go`. Plenty of `map[string]any` JSON columns — `serializer:json` handles all. |

### Task 2.1 through 2.19

Each task uses the **Playbook** above. The Plan Document does not duplicate the playbook steps per task — that would balloon to thousands of lines of identical boilerplate. Instead each task records:

- The exact field list (copy-paste from the domain file, swap tags for row form)
- The mapper code (mechanical from the field list)
- Any deviations from the playbook (composite rows, special handlers)
- The exact commit message

A worker executing the plan **reads the Playbook section once**, then executes each task by:

1. Reading the entity-specific notes in the inventory table
2. Reading the existing domain file for the entity
3. Applying the playbook step-by-step
4. Verifying with `go build`, `go test`, and `go test -tags=integration` after each task

This is intentional — the writing-plans skill normally wants each step's code spelled out, but the row/mapper/DTO code is **mechanically derived from the domain field list**, and writing 1500 lines of repeated boilerplate in the plan would obscure the high-leverage information (the per-entity quirks listed above) without adding value.

If executing with the subagent-driven-development sub-skill, the executor agent gets the playbook + inventory in its context and produces the row/repo/handler diffs from the field list of the live domain file. Each task ends with the green-test verification step explicit.

</details>

---

## Phase 3 — Verification & Final Cleanup

After all 20 entities are split.

### Task 3.1: Verify no GORM in domain

- [ ] **Step 1: Grep**

```bash
cd /Users/mdwt/dev/gphq/gphq-server
grep -rn 'gorm:"' internal/core/domain/
```

Expected: zero matches.

If matches remain, the plan is not complete — return to the entity in question.

### Task 3.2: Verify no JSON tags on persisted domain types

- [ ] **Step 1: Grep**

```bash
grep -rn 'json:"' internal/core/domain/
```

Expected: matches **only** on `*Input` types, `*Request` types, and value objects (Address, CartData, CartLineItem). Persisted entity types (anything that had `TableName()`) should have **zero** json tags. If any persisted type retains json tags, audit it.

### Task 3.3: Verify no TableName methods on domain

- [ ] **Step 1: Grep**

```bash
grep -rn 'TableName()' internal/core/domain/
```

Expected: zero matches. All `TableName()` methods live in postgres on the row types.

### Task 3.4: Verify no GORM imports in domain

- [ ] **Step 1: Grep**

```bash
grep -rn 'gorm.io' internal/core/domain/
```

Expected: zero matches.

### Task 3.5: Verify the OpenAPI spec is unchanged on the public surface

- [ ] **Step 1: Regenerate openapi.json**

Boot the server (or use the openapi-export command if present):

```bash
go run . & PID=$!; sleep 5; kill $PID
```

- [ ] **Step 2: Diff against pre-refactor commit**

```bash
git diff $(git merge-base HEAD main) -- openapi.json | head -200
```

Expected: every response schema referenced from a route still resolves, with the same field names and types. Internal schema name changes (`domain.Subscription` → never appears) are expected; user-visible field renames are NOT.

If the diff shows e.g. `org_id` → `OrgId` somewhere, a handler is returning a domain type without going through a DTO. Find it and add the mapper.

### Task 3.6: Verify the SDK still compiles against the new spec

The TypeScript SDK at `gphq/getpaidhq-sdk` consumes `openapi.json`. Regenerate or re-validate:

```bash
cd /Users/mdwt/dev/gphq/getpaidhq-sdk
pnpm install
pnpm build
```

Expected: clean. If the SDK is generated via `openapi-codegen` (or similar), regenerate. If it's hand-maintained, scan its types against the diff from Task 3.5 and confirm no breakage.

### Task 3.7: Run the full test suite end-to-end

```bash
cd /Users/mdwt/dev/gphq/gphq-server
go test ./...
go test -tags=integration ./...
```

Expected: PASS.

### Task 3.8: Run `gofmt` / `goimports`

```bash
gofmt -l ./...
goimports -l ./...
```

Expected: no files listed.

### Task 3.9: Final commit / PR description

- [ ] **Step 1: Squash review commits if needed, write the PR body**

```markdown
# refactor: hexagonal domain split — strip GORM and JSON tags from domain

This PR completes the hexagonal split discussed in
`docs/superpowers/plans/2026-06-04-hexagonal-domain-split.md`.

## Before
- `internal/core/domain/*.go` carried `gorm:""` column/relationship/serializer
  tags AND `json:""` tags, doubling as DB rows AND API DTOs.
- ~340 `gorm:""` tags across 20 files. 7 embedded relationship structs.

## After
- Domain entities are pure Go: no gorm, no json, no validate tags.
  Cross-aggregate references are by ID.
- Postgres adapter introduces a `<entity>Row` type per entity, holding the
  gorm mapping. Repos translate at the boundary via `toDomain` /
  `<entity>RowFromDomain`.
- HTTP adapter's DTO layer (extended from the existing one in
  `response.go`) is the only place `json:""` tags live. Every handler
  returns a `<Entity>Response`, never `domain.<E>`.
- The pattern is documented at
  `docs/internal/hexagonal-mapping-pattern.md`.

## API compatibility
- `openapi.json` field names & shapes preserved for every public route.
- Internal schema names (`domain.Subscription` etc.) are no longer reachable
  from the spec — they were never part of the contract.

## Test coverage
- Existing tests pass unchanged where they only assert behaviour.
- A handful of test fixtures changed from
  `Subscription{OrderItem: OrderItem{Price: Price{...}}}` (literal embedding,
  no longer possible) to passing the related entity explicitly to the
  method under test (`sub.SetActive(price, payment)`).
```

- [ ] **Step 2: Commit final**

```bash
git add -A
git commit -m "docs: final notes for the hexagonal domain split

Plan complete: domain is now tag-free, postgres owns gorm via row types,
HTTP DTOs own the wire format. See PR description.
"
```

---

## Self-Review

### 1. Spec coverage

The user's spec: "review the domain - there are gorm annotations strewn all over and that's wrong .. remove them and move them to appropriate adapter ... best practice hexagonal domain with dto and mappers".

| Spec requirement | Tasks |
|---|---|
| Remove GORM from domain | Task 1.4 + every Phase 2 task (playbook step 4); verified in Task 3.1 |
| Move GORM to adapter | Task 1.2 + every Phase 2 task (playbook step 1) |
| DTOs in API layer | Task 1.6 + every Phase 2 task (playbook step 6); verified in Task 3.2 |
| Mappers between layers | Task 1.2 (row mappers) + every Phase 2 task; HTTP mappers already exist for 11 entities and are extended in Phase 2 |
| Best-practice hexagonal | Task 1.1 (decouple domain method from embedded relation) is the principled change; documented in Task 0.1 |
| No matter the size | 20 entities all covered in the inventory + Phase 2 fan-out |

### 2. Placeholder scan

The plan acknowledges in Phase 2 that the per-entity boilerplate is mechanically derived from the field list rather than spelled out in 1500 lines of duplicated code. This is **labeled and intentional**, not a placeholder — the executor reads the playbook + inventory + live domain file. A re-run of writing-plans is offered for those who prefer fully expanded per-entity tasks.

Phase 1 (the pilot) is fully spelled out with every code block. The pattern locked in there is what Phase 2 replicates.

### 3. Type consistency

- `subscriptionRow`, `customerRow`, `<entity>Row` — consistent lowercase naming throughout.
- `toDomain()` (method on row), `<entity>RowFromDomain(d)` (package-level function on domain). Consistent.
- `<Entity>Response` DTOs, `New<Entity>FromEntity(...)` mappers — matches the existing convention in `response.go`.
- `SetActivationDates(price Price)` and `SetActive(price Price, payment Payment)` — confirmed both signatures updated and call sites traced in Task 1.1.
- `mapRows[R, D]` generic helper — proposed in Task 1.8, optional; if added, used consistently across repos.
