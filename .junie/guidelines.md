# GoLand Development Guidelines for Payloop

This file provides guidelines for Junie AI agent working within GoLand IDE on the Payloop billing platform.

## Project Structure and DDD Architecture

### Core Architecture Layers
```
payloop/
├── internal/
│   ├── api/           # HTTP API layer (controllers, routes, DTOs)
│   ├── application/   # Application services and interfaces
│   ├── domain/        # Domain entities, repositories, business logic
│   └── infrastructure/ # External service implementations
├── docs/              # Documentation
├── specs/             # Business specifications
└── prisma/           # Database schema and migrations
```

### Domain-Driven Design (DDD) Principles
- **Domain Layer**: Pure business logic, no infrastructure dependencies
- **Application Layer**: Orchestrates domain operations, implements use cases
- **Infrastructure Layer**: Database, external services, technical concerns
- **API Layer**: HTTP endpoints, request/response DTOs, validation

### DTO Layer Separation (CRITICAL)
**NEVER use API DTOs in Application Services - this violates clean architecture!**

#### API DTOs (`internal/api/dto/`)
- **Purpose**: HTTP request/response serialization only
- **Location**: `internal/api/dto/request/` and `internal/api/dto/response/`
- **Usage**: Only in controllers and API layer
- **Contains**: HTTP-specific fields, validation tags, JSON serialization

#### Application DTOs (`internal/application/dto/`)
- **Purpose**: Application service inputs/outputs
- **Location**: `internal/application/dto/`
- **Usage**: Application services and domain coordination
- **Contains**: Business logic parameters, domain-focused data structures

#### Mapping Pattern
```go
// ✅ CORRECT: Controller handles conversion
func (c *CustomerController) Create(ctx *gin.Context) {
    var apiReq request.CreateCustomerRequest
    if err := ctx.ShouldBindJSON(&apiReq); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Convert API DTO → Application DTO
    appInput := mappers.ToCreateCustomerInput(apiReq)
    
    // Application service uses application DTO
    customer, err := c.customerService.Create(ctx, orgId, appInput)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // Convert Domain Entity → API DTO
    response := mappers.ToCustomerResponse(customer)
    ctx.JSON(200, response)
}

// ❌ WRONG: Service uses API DTOs directly
func (s *CustomerService) Create(ctx context.Context, orgId string, req request.CreateCustomerRequest) (response.CustomerResponse, error) {
    // This violates clean architecture!
}
```

#### Service Interface Patterns
```go
// ✅ CORRECT: Application service interface
type CustomerService interface {
    Create(ctx context.Context, orgId string, input dto.CreateCustomerInput) (entities.Customer, error)
    List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Customer], error)
}

// ❌ WRONG: Using API DTOs in service interface
type CustomerService interface {
    Create(ctx context.Context, orgId string, req request.CreateCustomerRequest) (response.CustomerResponse, error)
    List(ctx context.Context, orgId string, pagination request.Pagination) (response.CustomerListResponse, error)
}
```

## Testing Patterns

- use existing shared mocks and fixtures in `internal/testing`, or add new ones as needed in internal/testing

### Table-Driven Test Template
```go
func TestEntity_SomeMethod(t *testing.T) {
    tests := []struct {
        name        string
        input       InputType
        expected    ExpectedType
        expectError bool
        errorMsg    string
    }{
        {
            name:     "valid input",
            input:    InputType{/* valid data */},
            expected: ExpectedType{/* expected result */},
            expectError: false,
        },
        {
            name:        "invalid input",
            input:       InputType{/* invalid data */},
            expectError: true,
            errorMsg:    "validation error expected",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := SomeMethod(tt.input)
            
            if tt.expectError {
                assert.Error(t, err)
                if tt.errorMsg != "" {
                    assert.Contains(t, err.Error(), tt.errorMsg)
                }
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Repository Test Template
```go
func TestEntityRepository_Create(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()
    
    repo := NewEntityRepository(db, logger.NewNopLogger())
    
    // Create test entity
    entity := entities.Entity{
        OrgId: "test-org",
        Id:    "test-id",
        // ... other fields
    }
    
    // Test creation
    created, err := repo.Create(context.Background(), entity)
    assert.NoError(t, err)
    assert.Equal(t, entity.Id, created.Id)
    assert.NotZero(t, created.CreatedAt)
}
```

## File Organization Conventions

### Naming Conventions
- **Files**: `snake_case.go`
- **Types**: `PascalCase`
- **Functions**: `PascalCase` for exported, `camelCase` for private
- **Constants**: `PascalCase` with prefix
- **Variables**: `camelCase`


### Error Handling Patterns
```go
// Domain errors
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// Infrastructure error wrapping
func (r Repository) SomeMethod(ctx context.Context) error {
    err := r.db.Query(ctx, query)
    if err != nil {
        r.logger.Error("database query failed", err.Error())
        return fmt.Errorf("failed to execute query: %w", err)
    }
    return nil
}
```

## Multi-tenancy Implementation

### OrgId Enforcement
```go
// Always include orgId in repository methods
func (r Repository) FindById(ctx context.Context, orgId, id string) (Entity, error) {
    query := `SELECT * FROM table WHERE org_id = @org_id AND id = @id`
    // ...
}

// Service methods always start with orgId
func (s Service) Get(ctx context.Context, orgId, id string) (Entity, error) {
    return s.repository.FindById(ctx, orgId, id)
}

// Controller extracts orgId from authenticated user
func (c Controller) Get(ctx *gin.Context) {
    user, _ := ctx.Get("user")
    authUser := user.(authn.User)
    
    entity, err := c.service.Get(ctx.Request.Context(), authUser.OrgId, id)
    // ...
}
```


### Temporal Activity Pattern
```go
// Activities should be thin coordinators
func (a *BillingActivity) ProcessSubscriptionCharge(ctx context.Context, subscriptionId string) error {
    // Delegate to domain service - NO business logic here
    return a.subscriptionService.ProcessCharge(ctx, subscriptionId)
}

// Business logic stays in domain services
func (s *SubscriptionService) ProcessCharge(ctx context.Context, subscriptionId string) error {
    // Complex business logic here
    subscription, err := s.repository.FindById(ctx, subscriptionId)
    // ... billing logic
    return nil
}
```

### Common Workflow Patterns
- **Order Processing**: Cart → Order → Payment → Subscription activation
- **Subscription Charging**: Usage aggregation → Invoice generation → Payment processing
- **Dunning Process**: Payment failure → Retry schedule → Communication → Resolution

## Database Patterns

### Migration Strategy
- Use Prisma for schema management
- Dont generate migrations manually, use `prisma migrate dev`
- Keep migrations atomic and reversible
- Test migrations on copy of production data
- Use transactions for multi-table changes

### Query Optimization
- Always include `org_id` in WHERE clauses
- Use appropriate indexes for common queries
- Limit result sets with pagination
- Use prepared statements for repeated queries


This guideline document provides comprehensive patterns and templates for implementing features in the Payloop billing platform while maintaining DDD principles and code quality standards.