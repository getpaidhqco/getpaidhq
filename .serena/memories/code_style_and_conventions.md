# Payloop Code Style and Conventions

## Domain-Driven Design (DDD) Principles
- **Domain Layer Purity**: Keep domain entities free of infrastructure concerns
- **Entity Factories**: Place entity creation logic in domain layer (`internal/domain/entities/`)
- **Repository Interfaces**: Define in domain layer, implement in infrastructure layer
- **Value Objects**: Use for complex data types with validation rules
- **Aggregate Roots**: Maintain consistency boundaries within aggregates

## Critical DTO Layer Separation Rules
**NEVER use API DTOs in Application Services - this is a fundamental clean architecture violation!**

### DTO Usage Rules:
1. **API DTOs** (`internal/api/dto/request` & `internal/api/dto/response`)
   - ONLY used in controllers and API layer
   - Handle HTTP serialization, validation tags, API-specific formatting
   - FORBIDDEN in application services, domain layer, or infrastructure

2. **Application DTOs** (`internal/application/dto/`)
   - Used in application service interfaces and implementations
   - Business-focused data structures without HTTP concerns
   - Bridge between API layer and domain layer

3. **Domain Entities**
   - Pure business objects with no external dependencies
   - Returned by application services and repositories
   - Never contain serialization or HTTP-specific logic

## Function Design Guidelines
- Function calls should return structs, not pointers (when possible)
- Reference adjacent files for examples of implementation patterns
- Use existing codebase patterns as templates for new features
- Prefer composition over inheritance
- Keep functions focused on single responsibilities
- Prefer structs over pointers for data structures

## Entity Construction Patterns
```go
// Always use factory methods for entities with validation
price, err := entities.NewPrice(orgId, variantId, input)
if err != nil {
    return err // Handle validation errors
}

// Use constructors for subscription items
subscriptionItem := entities.NewSubscriptionItem(orgId, subscriptionId, priceId, description, currency)
```

## Multi-tenancy
- Always include `orgId` in queries and entities
- Enforce tenant isolation at repository level
- Use `orgId` as first parameter in service methods
- Validate tenant access in controllers

## Error Handling
- Use domain-specific error types
- Wrap infrastructure errors appropriately
- Maintain error context through layers
- Log errors at infrastructure boundaries