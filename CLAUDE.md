# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Always examine existing files and project structure when generating files. Use the same coding style, dependencies, layout, file structure etc. for the DDD project. Follow the Domain-Driven Design (DDD) principles and clean architecture patterns used in this project.

ALWAYS give the best solution, even if it requires more code or database changes. The goal is to maintain a clean, maintainable, and scalable codebase.

## Core Development Principles

### Domain-Driven Design (DDD) Adherence
- **Domain Layer Purity**: Keep domain entities free of infrastructure concerns
- **Entity Factories**: Place entity creation logic in domain layer (`internal/domain/entities/`)
- **Repository Interfaces**: Define in domain layer, implement in infrastructure layer
- **Value Objects**: Use for complex data types with validation rules
- **Aggregate Roots**: Maintain consistency boundaries within aggregates

### Code Organization Rules
- **No Cross-Layer Dependencies**: Infrastructure cannot import domain, application cannot import infrastructure
- **DTO Placement**: API DTOs belong in `internal/api/dto/`, Application DTOs in `internal/application/dto/`
- **Mapping Logic**: Use dedicated mappers to convert between DTO layers
- **Entity Construction**: Always use factory methods or constructors for complex entities
- **Validation**: Implement validation in entity constructors and value objects

### DTO Layer Separation Rules (CRITICAL - NO EXCEPTIONS)

**NEVER use API DTOs in Application Services - this is a fundamental clean architecture violation!**

#### Strict DTO Usage Rules:

1. **API DTOs** (`internal/api/dto/request` & `internal/api/dto/response`)
   - **ONLY** used in controllers and API layer
   - Handle HTTP serialization, validation tags, API-specific formatting
   - **FORBIDDEN** in application services, domain layer, or infrastructure

2. **Application DTOs** (`internal/application/dto/`)
   - Used in application service interfaces and implementations
   - Business-focused data structures without HTTP concerns
   - Bridge between API layer and domain layer

3. **Domain Entities**
   - Pure business objects with no external dependencies
   - Returned by application services and repositories
   - Never contain serialization or HTTP-specific logic

#### Required Patterns When Creating New Features:

```go
// ✅ CORRECT Pattern - Controller with proper mapping
func (c *CustomerController) Create(ctx *gin.Context) {
    // 1. Parse API DTO
    var apiReq request.CreateCustomerRequest
    if err := ctx.ShouldBindJSON(&apiReq); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 2. Convert API DTO → Application DTO  
    appInput := mappers.ToCreateCustomerInput(apiReq)
    
    // 3. Call application service with application DTO
    customer, err := c.customerService.Create(ctx, orgId, appInput)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 4. Convert Domain Entity → API DTO
    response := mappers.ToCustomerResponse(customer)
    ctx.JSON(200, response)
}

// ✅ CORRECT - Application Service Interface
type CustomerService interface {
    Create(ctx context.Context, orgId string, input dto.CreateCustomerInput) (entities.Customer, error)
    List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Customer], error)
}

// ✅ CORRECT - Application DTO
type CreateCustomerInput struct {
    Email          string            `json:"email"`
    FirstName      string            `json:"first_name"`
    LastName       string            `json:"last_name"`
    BillingAddress *entities.Address `json:"billing_address,omitempty"`
    Metadata       map[string]string `json:"metadata,omitempty"`
}

// ❌ WRONG - Service using API DTOs
type CustomerService interface {
    Create(ctx context.Context, orgId string, req request.CreateCustomerRequest) (response.CustomerResponse, error)
    List(ctx context.Context, orgId string, pagination request.Pagination) (response.CustomerListResponse, error)
}

// ❌ WRONG - Application service importing API DTOs
import (
    "payloop/internal/api/dto/request"  // NEVER import this in application layer
    "payloop/internal/api/dto/response" // NEVER import this in application layer
)
```

#### Mapping Functions Pattern:
Create mappers in `internal/api/mappers/` to handle conversions:

```go
// ToCreateCustomerInput converts API request to application input
func ToCreateCustomerInput(req request.CreateCustomerRequest) dto.CreateCustomerInput {
    return dto.CreateCustomerInput{
        Email:          req.Email,
        FirstName:      req.FirstName,
        LastName:       req.LastName,
        BillingAddress: req.BillingAddress,
        Metadata:       req.Metadata,
    }
}

// ToCustomerResponse converts domain entity to API response  
func ToCustomerResponse(customer entities.Customer) response.CustomerResponse {
    return response.CustomerResponse{
        Id:             customer.Id,
        Email:          customer.Email,
        FirstName:      customer.FirstName,
        LastName:       customer.LastName,
        BillingAddress: customer.BillingAddress,
        CreatedAt:      customer.CreatedAt,
        UpdatedAt:      customer.UpdatedAt,
        Metadata:       customer.Metadata,
    }
}
```

#### File Creation Rules:
- When creating new services: **ALWAYS** use application DTOs, **NEVER** API DTOs
- When creating new controllers: **ALWAYS** include mapping between API and application DTOs
- When creating new features: Create application DTOs first, then API DTOs, then mappers

### Function Design Guidelines
- **Function calls should return structs, not pointers (when possible)**
- **ALWAYS prefer structs over pointers for data structures**
  - Use structs for DTOs, response objects, and data containers
  - Only use pointers when necessary for performance (large structs) or when nil semantics are required
  - For optional fields in structs, use zero values and omitempty tags instead of pointers
  - Examples:
    ```go
    // ✅ CORRECT - Use struct
    type PublicCustomer struct {
        Email     string `json:"email"`
        FirstName string `json:"first_name,omitempty"` // Empty string for optional
    }
    
    // ❌ WRONG - Avoid pointers unless necessary
    type PublicCustomer struct {
        Email     *string `json:"email"`
        FirstName *string `json:"first_name,omitempty"`
    }
    ```
- Reference adjacent files for examples of implementation patterns
- Use existing codebase patterns as templates for new features
- Prefer composition to inheritance
- Keep functions focused on single responsibilities

### Struct vs Pointer Usage Guidelines

**Default Rule: ALWAYS use structs over pointers unless there's a specific need for pointers**

#### When to Use Structs (Preferred):
- **DTOs and Response Objects**: All API response DTOs should use structs
- **Optional Fields**: Use `omitempty` JSON tags with zero values instead of pointers
- **Configuration Objects**: Structs with default zero values
- **Small to Medium Data Structures**: Most business data structures
- **Function Parameters**: Pass structs by value for small-medium structures

#### When Pointers Are Acceptable:
- **Large Structs**: When copying would be expensive (>1KB as rough guideline)
- **Nil Semantics Required**: When you need to distinguish between "not set" and "zero value"
- **Interface Implementation**: When required for interface satisfaction
- **Receiver Methods**: Use pointer receivers for methods that modify the struct
- **Database Models**: When ORM requires pointer fields

#### Examples:

```go
// ✅ PREFERRED - Struct with omitempty for optional fields
type CreateCustomerRequest struct {
    Email     string `json:"email" binding:"required"`
    FirstName string `json:"first_name,omitempty"`
    LastName  string `json:"last_name,omitempty"`
}

// ✅ ACCEPTABLE - Pointer when nil semantics needed
type UpdateCustomerRequest struct {
    Email     *string `json:"email,omitempty"` // nil = don't update, "" = set empty
    FirstName *string `json:"first_name,omitempty"`
}

// ❌ AVOID - Unnecessary pointers for simple fields
type BadCustomerRequest struct {
    Email     *string `json:"email"`
    FirstName *string `json:"first_name"`
}
```

## Docs and Specs
- Documentation is in `docs/`
- Specifications are in `specs/`
- Usage-based billing specs are in `specs/usage-types.md`

### Building and Running
- `go run main.go serve` - Start the API server
- `docker-compose up -d` - Start required services (database, temporal, etc.)

### Database Operations
- `pnpm dlx prisma generate` - Generate Prisma client
- `pnpm dlx prisma db push` - Push schema changes to development database
- `pnpm dlx prisma migrate deploy` - Deploy migrations (used in CI/CD)
- `pnpm dlx prisma format` - Format Prisma schema files

### Reporting Database
- `pnpm dlx prisma format --schema=schemas/reporting/schema.prisma` - Format reporting schema
- `pnpm dlx prisma db push --schema=schemas/reporting/schema.prisma` - Push reporting schema changes

### Testing
- `go test ./...` - Run all tests
- `go test ./internal/application/services/...` - Run service layer tests
- `go test -v ./internal/application/lib/pdf/...` - Run PDF generation tests with verbose output

### Deployment
- `pnpm run deploy:test` - Deploy to test environment
- `pnpm run deploy:prod` - Deploy to production environment

### Development Tunnels
- `pnpm run tunnel:test` - Create SSH tunnel to test environment resources
- `pnpm run tunnel:prod` - Create SSH tunnel to production environment resources

## Architecture Overview

Payloop is a subscription billing platform built using Domain-Driven Design (DDD) with clean architecture principles:

### Core Layers
1. **API Layer** (`internal/api/`) - HTTP controllers, routes, middlewares, DTOs
2. **Application Layer** (`internal/application/`) - Business logic services and interfaces
3. **Domain Layer** (`internal/domain/`) - Core business entities, repositories, value objects
4. **Infrastructure Layer** (`internal/infrastructure/`) - External service implementations

### Key Technologies
- **Web Framework**: Gin (HTTP routing)
- **Database**: PostgreSQL with Prisma ORM for schema management
- **Dependency Injection**: Uber FX
- **Workflow Engine**: Temporal for complex business workflows
- **Pub/Sub**: NATS for event messaging
- **Cache**: Redis for caching
- **Queue**: AWS SQS for job processing
- **Authentication**: Clerk (enabled), Cognito, API Keys (configurable in modules.go:35-37)
- **Authorization**: Cedar policy engine

### Important Patterns

#### Module System
- Each infrastructure component has its own module in `internal/infrastructure/`
- Modules are registered in `internal/application/bootstrap/modules.go`
- Authentication providers can be enabled/disabled by commenting/uncommenting in modules.go

#### Database Architecture
- **Main Database** (`payloop`): Operational data
- **Reporting Database** (`payloop_reporting`): Analytics and reporting
- **CDC Sync**: Change Data Capture keeps databases synchronized
- **Schema Location**: Main schema in `prisma/schema.prisma`, reporting in `schemas/reporting/schema.prisma`

#### Workflow Orchestration
- Complex business processes use Temporal workflows in `internal/infrastructure/workflow/temporal/workflows/`
- Activities are defined in `internal/infrastructure/workflow/temporal/activities/`
- Keep Business Logic in Domain Services
  - Activities should be **thin coordinators** that delegate to domain services
  - Activities should never include orchastration services (e.g. SubscriptionOrchestrationService), it should 
  delegate to domain services like SubscriptionService, PaymentService, etc.
  - Domain services contain the actual business logic and maintain DDD principles
  - This preserves testability and domain purity


#### Testing Structure
- Unit tests alongside source files (e.g., `service_test.go`)
- Integration tests in dedicated files (e.g., `pdf_integration_test.go`)
- Test database seeding available via `prisma/seed.js`

## Key Business Domains

### Subscription Management
- **Entities**: Subscription, Customer, Payment, Invoice
- **Workflows**: Subscription charging, pause/resume, cancellation
- **Recovery**: Automatic retry logic for failed payments

### Payment Processing
- **Providers**: Paystack (active), Checkout.com (configured)
- **Features**: Payment method management, refunds, webhooks
- **Location**: `internal/infrastructure/payments/`

### Invoice Generation
- **PDF Generation**: Uses chromedp (headless Chrome) via `internal/application/lib/pdf/`
- **Templates**: Liquid templates in `assets/templates/invoices/`
- **Formats**: Multiple invoice template variations

### Event System
- **Topics**: Defined in `internal/application/lib/events/topic/`
- **Webhooks**: Outgoing webhook system for external integrations
- **Queue Processing**: SQS-based job processing

## Usage-Based Billing Implementation

### Pricing Model Categories
- **Traditional**: Fixed recurring subscription amounts
- **Usage-Based**: Pure usage billing with unit pricing, percentage fees, or transaction fees
- **Hybrid**: Fixed base amount + usage-based overage charges

### Usage Types and Implementation
- **API Calls**: Count-based billing with aggregation (sum, max, last_during_period)
- **Data Transfer**: Volume-based with unit pricing per GB/MB
- **Transaction Fees**: Percentage or fixed fee per transaction
- **Active Users**: Tiered pricing based on user count
- **Storage**: Volume-based with aggregation types

### Key Implementation Files
- **Entities**: `internal/domain/entities/price.go`, `internal/domain/entities/subscription_item.go`
- **Usage Records**: `internal/domain/entities/usage_record.go`
- **Repository**: `internal/infrastructure/db/postgres/usage_record_repository.go`
- **Types**: `internal/domain/entities/usage_types.go`

### Entity Construction Patterns
```go
// Always use factory methods for entities with validation
price, err := entities.NewPrice(orgId, variantId, input)
if err != nil {
    return err // Handle validation errors
}

// Use constructors for subscription items
subscriptionItem := entities.NewSubscriptionItem(orgId, subscriptionId, priceId, description, currency)
```

### Test Strategy
- **Table-Driven Tests**: Use for multiple scenarios
- **Mock Interfaces**: Repository interfaces, not implementations
- **Test Coverage**: Include traditional, usage-based, hybrid, and edge cases
- **Validation Testing**: Test entity validation rules comprehensively

## Temporal Workflow Patterns

### Activity Design Rules
- **Thin Coordinators**: Activities should delegate to domain services
- **No Orchestration**: Never include orchestration services in activities
- **Domain Service Delegation**: Activities call domain services (SubscriptionService, PaymentService)
- **Business Logic Placement**: Keep business logic in domain services, not activities

### Workflow Structure
```go
// Good: Activity delegates to domain service
func (a *BillingActivity) ProcessSubscriptionCharge(ctx context.Context, subscriptionId string) error {
    return a.subscriptionService.ProcessCharge(ctx, subscriptionId)
}

// Bad: Activity contains business logic
func (a *BillingActivity) ProcessSubscriptionCharge(ctx context.Context, subscriptionId string) error {
    // Complex billing logic here - belongs in domain service
}
```

## Configuration Notes

### Authentication Setup
To change authentication providers, modify `internal/application/bootstrap/modules.go`:
- Uncomment desired auth modules (lines 35-37)
- Only one auth provider should be active at a time

### Environment Configuration
- Base config in `config.yml`
- Override with environment variables following standard patterns
- Database connections configured per environment

### CDC Troubleshooting
When CDC sync fails, manually reset replication:
```sql
DROP PUBLICATION cdc_pub;
SELECT pg_drop_replication_slot('cdc_slot2');
```

## Development Setup Requirements

1. **Prerequisites**: Docker, Go 1.24, Node.js/pnpm for Prisma
2. **Services**: Start with `docker-compose up -d`
3. **Temporal**: Create namespace with `temporal operator namespace create -n subscriptions`
4. **Database**: Run migrations and seed data
5. **Configuration**: Copy and modify `config.yml` for local environment

## Common Implementation Patterns

### Entity Validation
```go
func NewPrice(orgId, variantId string, input CreatePriceInput) (Price, error) {
    if err := validatePriceInput(input); err != nil {
        return Price{}, err
    }
    
    price := Price{
        OrgId:     orgId,
        VariantId: variantId,
        // ... set fields
    }
    
    return price, nil
}
```

### Repository Pattern
```go
// Define interface in domain layer
type SubscriptionRepository interface {
    Create(ctx context.Context, subscription entities.Subscription) (entities.Subscription, error)
    FindById(ctx context.Context, orgId, id string) (entities.Subscription, error)
}

// Implement in infrastructure layer
type subscriptionRepository struct {
    *PgDatabase
}
```

### Error Handling
- Use domain-specific error types
- Wrap infrastructure errors appropriately
- Maintain error context through layers
- Log errors at infrastructure boundaries

### Multi-tenancy
- Always include `orgId` in queries and entities
- Enforce tenant isolation at repository level
- Use `orgId` as first parameter in service methods
- Validate tenant access in controllers

## Documentation Page Format Requirements

When creating documentation pages (*.mdx files), always follow this structure:

### Frontmatter Requirements
Every documentation page MUST include frontmatter with `title` and `description`:

```markdown
---
title: Page Title
description: Concise description of what this page covers (1-2 sentences, under 160 characters)
---

# Page Title

Page content starts here...
```

### Page Structure Guidelines
- **Title**: Use the same title in frontmatter and H1 heading
- **Description**: Should be SEO-friendly, descriptive, and under 160 characters
- **H1**: Only one H1 per page (the main title)
- **Content**: Use clear headings (H2, H3) for organization
- **Code examples**: Include practical examples with proper syntax highlighting
- **Cross-references**: Link to related pages and concepts

### Examples of Good Descriptions
- API pages: "Manage customer data, payment methods, balances, and portal sessions with comprehensive CRUD operations"
- Guide pages: "Learn the core concepts and implementation patterns for usage-based billing in GetPaidHQ"
- Reference pages: "Complete reference for webhook events, payload structure, and verification methods"

### File Naming Convention
- Use kebab-case for file names (e.g., `rate-limits.mdx`, `getting-started.mdx`)
- Match navigation structure defined in meta.json files
- Keep file names concise but descriptive