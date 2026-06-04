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
│                            • Read Models (composed query results)        │
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
| Domain service | `internal/core/domain/` | none | pure functions or methods that encode business rules |
| Domain event | `internal/core/domain/` | none | `OrderCompletedEvent` |
| Command / Query Input | `internal/core/service/` | none | `service.CreateSubscriptionInput` |
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
   customer is a use-case concern.

6. **Domain methods take what they need explicitly.** If `Subscription.SetActive`
   needs a `Price`, the price is a parameter, not something assumed to be
   loaded on `self`. This makes the method independent of how the entity was
   constructed and prevents implicit "the relation must be hydrated" coupling.

7. **Aggregates own their invariants.** Methods that mutate state validate
   internal consistency.

8. **Transient fields don't belong on the entity.** If a field is populated
   only at creation (e.g. a raw API key returned once), or is derived from
   other fields (e.g. a cart total), it does not belong as a struct field on
   the domain entity. Use a service result type or a method.

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
   - Only top-level GET endpoints earn a read model. Sub-entities loaded as
     part of a parent query are sub-types inside the parent read model, not
     top-level.
   - List endpoints reuse the same read model (`ListDetails([]Details)`)
     unless a real list-specific shape demands its own model.

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
   domain entities; a row type is package-internal. A handler accepts request
   DTOs and returns response DTOs; the domain entity is internal to it.

## Postgres adapter specifics

- Row types are **lowercase** (`subscriptionRow`) — internal to the package.
- `TableName()` lives on the row.
- GORM relationship tags MAY exist on rows when a composite `Preload` query
  is the cheapest correct shape inside a repo method. The repo still returns
  a single aggregate to its caller; composition happens in the service layer.
- Mappers: `(r row) toDomain() domain.Entity` and
  `entityRowFromDomain(e domain.Entity) row`.
- Repos referenced inside read-model composition gain a batched
  `FindByIds(ctx, orgId, ids []string) ([]domain.E, error)` method to prevent
  N+1 in list endpoints.

## HTTP adapter specifics

- Request DTO names: `<Action><Entity>Request` (e.g. `CreateSubscriptionRequest`).
- Response DTO names: `<Entity>Response` (e.g. `SubscriptionResponse`).
- Request DTOs carry `validate:""` tags and a `.ToInput(orgId string) service.X`
  method (orgId comes from `AuthUserFrom(c)` at the handler).
- Response DTOs carry `json:""` tags only. Mapper:
  `NewEntityFromEntity(e domain.E) EntityResponse`, and where a read model
  exists, `NewEntityResponseFromDetails(d service.EntityDetails) EntityResponse`.
- Nested response DTOs are built inline via the nested mapper:
  `Customer: NewCustomerFromEntity(c)`.
- Handlers NEVER return `domain.X` directly — that would leak the
  (intentionally tag-free) domain type and produce `OrgId` instead of `org_id`
  in the JSON output.
- File layout is per-entity: `<entity>_request.go`, `<entity>_response.go`,
  `<entity>_handler.go`. (Legacy: `internal/adapter/http/response.go` will be
  split into per-entity files during the refactor.)

## Workflow adapter specifics (Hatchet / Temporal)

- Workflow step inputs are serialized over the durable log as JSON.
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
4. If the entity has a nested response shape, define the **read model** in
   `internal/core/service/<entity>_read.go` and the `GetDetails` /
   `ListDetails` query handler methods on the service.
5. Define the **port** for its repository in `internal/core/port/`. Include
   `FindByIds` if any other entity's read model references this one.
6. Implement the **postgres row** at `internal/adapter/postgres/<entity>_row.go`
   with `toDomain` and `<entity>RowFromDomain` mappers.
7. Implement the **repo** at `internal/adapter/postgres/<entity>_repo.go`. Use
   the row type internally; translate at the boundary.
8. Add **HTTP DTOs** in `internal/adapter/http/<entity>_request.go` and
   `internal/adapter/http/<entity>_response.go`.
9. Implement the **HTTP handler** in `internal/adapter/http/<entity>_handler.go`.
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
