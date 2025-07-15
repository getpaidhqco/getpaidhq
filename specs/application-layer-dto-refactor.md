# Application Layer DTO Refactor Specification

## Overview

This specification outlines the refactoring required to fix clean architecture violations in the Payloop application layer, where API DTOs are incorrectly being used in application services. The goal is to establish proper separation of concerns between API and application layers.

## Problem Statement

### Current Issues
- Application services directly depend on API DTOs (`internal/api/dto/request` and `internal/api/dto/response`)
- This violates Domain-Driven Design (DDD) clean architecture principles
- Creates tight coupling between API and application layers
- Makes testing and maintenance more difficult
- Inconsistent patterns across the codebase

### Clean Architecture Principles
- **Application Layer** should only depend on domain entities and application-specific DTOs
- **API Layer** should handle conversion between API DTOs and application DTOs
- **Domain Layer** should remain pure with no external dependencies

## Scope

### Services Requiring Refactoring (Priority Order)

#### **Priority 1: Critical Violations**

1. **UsageRecordingService** (Complete Refactor Required)
   - **Files**: 
     - `/internal/application/interfaces/usage_recording_service.go`
     - `/internal/application/services/usage_recording_service.go`
   - **Issue**: All methods use API DTOs for input/output
   - **Impact**: High - Core billing functionality

2. **CustomerService** (Moderate Refactor Required)
   - **Files**:
     - `/internal/application/interfaces/customers.go`
     - `/internal/application/services/customer_service.go`
   - **Issue**: Input types embed API DTOs
   - **Impact**: High - Customer management

#### **Priority 2: Moderate Violations**

3. **SubscriptionService**
   - **Files**:
     - `/internal/application/interfaces/subscriptions.go`
     - `/internal/application/services/subscription_service.go`
   - **Issue**: Uses API pagination DTOs
   - **Impact**: Medium

4. **PaymentService**
   - **Files**:
     - `/internal/application/interfaces/payment.go`
     - `/internal/application/services/payment_service.go`
   - **Issue**: Uses API pagination and input DTOs
   - **Impact**: Medium

#### **Priority 3: Minor Violations**

5. **OrderService, DunningService, InvoiceService**
   - **Issue**: Primarily pagination-related violations
   - **Impact**: Low

## Implementation Plan

### Phase 1: Create Application DTOs

Create new application DTO files in `/internal/application/dto/`:

#### 1.1 Common DTOs (`common.go`)
```go
package dto

// Pagination represents application-layer pagination parameters
type Pagination struct {
    Page          int    `json:"page"`
    Limit         int    `json:"limit"`
    Offset        int    `json:"offset"`
    SortDirection string `json:"sort_direction"`
    SortBy        string `json:"sort_by"`
}

// PaginatedResult represents a paginated result set
type PaginatedResult[T any] struct {
    Items      []T `json:"items"`
    TotalCount int `json:"total_count"`
    Page       int `json:"page"`
    PageSize   int `json:"page_size"`
    HasMore    bool `json:"has_more"`
}
```

#### 1.2 Customer DTOs (`customers.go`)
```go
package dto

import (
    "payloop/internal/domain/entities"
    "payloop/internal/domain/entities/payment_methods"
)

// CreateCustomerInput represents input for creating a customer
type CreateCustomerInput struct {
    Email          string                 `json:"email"`
    FirstName      string                 `json:"first_name"`
    LastName       string                 `json:"last_name"`
    BillingAddress *entities.Address      `json:"billing_address,omitempty"`
    Phone          string                 `json:"phone,omitempty"`
    Metadata       map[string]string      `json:"metadata,omitempty"`
}

// UpdateCustomerInput represents input for updating a customer
type UpdateCustomerInput struct {
    Email          *string                `json:"email,omitempty"`
    FirstName      *string                `json:"first_name,omitempty"`
    LastName       *string                `json:"last_name,omitempty"`
    BillingAddress *entities.Address      `json:"billing_address,omitempty"`
    Phone          *string                `json:"phone,omitempty"`
    Metadata       map[string]string      `json:"metadata,omitempty"`
}

// CreatePaymentMethodInput represents input for creating a payment method
type CreatePaymentMethodInput struct {
    CustomerId     string                              `json:"customer_id"`
    Psp            string                              `json:"psp"`
    Name           string                              `json:"name"`
    Type           payment_methods.PaymentMethodType   `json:"type"`
    Details        interface{}                         `json:"details"`
    Token          string                              `json:"token,omitempty"`
    IsDefault      bool                                `json:"is_default"`
    BillingAddress *entities.Address                   `json:"billing_address,omitempty"`
    Metadata       map[string]string                   `json:"metadata,omitempty"`
}

// UpdatePaymentMethodInput represents input for updating a payment method
type UpdatePaymentMethodInput struct {
    Name           *string                `json:"name,omitempty"`
    IsDefault      *bool                  `json:"is_default,omitempty"`
    BillingAddress *entities.Address      `json:"billing_address,omitempty"`
    Metadata       map[string]string      `json:"metadata,omitempty"`
}
```

#### 1.3 Usage DTOs (`usage.go`)
```go
package dto

import (
    "time"
    "payloop/internal/domain/entities"
)

// RecordUsageInput represents input for recording usage
type RecordUsageInput struct {
    SubscriptionItemId string                 `json:"subscription_item_id"`
    Quantity          float64                `json:"quantity"`
    TransactionValue  *int64                 `json:"transaction_value,omitempty"`
    PercentageRate    *float64               `json:"percentage_rate,omitempty"`
    ReferenceId       string                 `json:"reference_id,omitempty"`
    ReferenceType     string                 `json:"reference_type,omitempty"`
    Timestamp         time.Time              `json:"timestamp"`
    Metadata          map[string]string      `json:"metadata,omitempty"`
}

// BatchRecordUsageInput represents input for batch recording usage
type BatchRecordUsageInput struct {
    Records []RecordUsageInput `json:"records"`
}

// UsageSummaryInput represents input for getting usage summary
type UsageSummaryInput struct {
    SubscriptionItemId string    `json:"subscription_item_id"`
    StartDate         time.Time `json:"start_date"`
    EndDate           time.Time `json:"end_date"`
}

// UsageSummaryResult represents usage summary data
type UsageSummaryResult struct {
    SubscriptionId     string                 `json:"subscription_id"`
    SubscriptionItemId string                 `json:"subscription_item_id"`
    BillingPeriod      string                 `json:"billing_period"`
    UsageType          entities.UsageType     `json:"usage_type"`
    UnitType           entities.UnitType      `json:"unit_type"`
    AggregationType    entities.AggregationType `json:"aggregation_type"`
    TotalQuantity      float64                `json:"total_quantity"`
    TotalAmount        int64                  `json:"total_amount"`
    Details            map[string]interface{} `json:"details"`
}

// ListUsageRecordsInput represents input for listing usage records
type ListUsageRecordsInput struct {
    SubscriptionItemId string     `json:"subscription_item_id"`
    Pagination        Pagination `json:"pagination"`
}

// GetSubscriptionUsageInput represents input for getting subscription usage
type GetSubscriptionUsageInput struct {
    SubscriptionId string    `json:"subscription_id"`
    StartDate     time.Time `json:"start_date"`
    EndDate       time.Time `json:"end_date"`
}
```

#### 1.4 Payment DTOs (`payments.go`)
```go
package dto

// RefundPaymentInput represents input for refunding a payment
type RefundPaymentInput struct {
    Amount int64  `json:"amount"`
    Reason string `json:"reason,omitempty"`
}

// ProcessPaymentInput represents input for processing a payment
type ProcessPaymentInput struct {
    Amount         int64             `json:"amount"`
    Currency       string            `json:"currency"`
    PaymentMethodId string           `json:"payment_method_id"`
    Description    string            `json:"description,omitempty"`
    Metadata       map[string]string `json:"metadata,omitempty"`
}
```

### Phase 2: Update Service Interfaces

#### 2.1 UsageRecordingService Interface
```go
package interfaces

import (
    "context"
    "time"
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
)

type UsageRecordingService interface {
    // Record single usage event
    RecordUsage(ctx context.Context, orgId string, input dto.RecordUsageInput) (entities.UsageRecord, error)

    // Record multiple usage events in batch
    BatchRecordUsage(ctx context.Context, orgId string, input dto.BatchRecordUsageInput) ([]entities.UsageRecord, error)

    // Get usage records with pagination
    ListUsageRecords(ctx context.Context, orgId string, input dto.ListUsageRecordsInput) (dto.PaginatedResult[entities.UsageRecord], error)

    // Get specific usage record
    GetUsageRecord(ctx context.Context, orgId string, usageRecordId string) (entities.UsageRecord, error)

    // Get usage summary for subscription item
    GetUsageSummary(ctx context.Context, orgId string, input dto.UsageSummaryInput) (dto.UsageSummaryResult, error)

    // Get subscription usage by billing period
    GetSubscriptionUsage(ctx context.Context, orgId string, input dto.GetSubscriptionUsageInput) ([]entities.UsageRecord, error)

    // Delete usage record (for corrections)
    DeleteUsageRecord(ctx context.Context, orgId string, usageRecordId string) error
}
```

#### 2.2 CustomerService Interface
```go
package interfaces

import (
    "context"
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
)

type CustomerService interface {
    // Customer operations
    Create(ctx context.Context, orgId string, input dto.CreateCustomerInput) (entities.Customer, error)
    Update(ctx context.Context, orgId string, customerId string, input dto.UpdateCustomerInput) (entities.Customer, error)
    Get(ctx context.Context, orgId string, id string) (entities.Customer, error)
    List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Customer], error)

    // Payment method operations
    CreatePaymentMethod(ctx context.Context, orgId string, input dto.CreatePaymentMethodInput) (entities.PaymentMethod, error)
    UpdatePaymentMethod(ctx context.Context, orgId string, paymentMethodId string, input dto.UpdatePaymentMethodInput) (entities.PaymentMethod, error)
    GetPaymentMethod(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error)

    // Secure payment method operations
    GetSecurePaymentMethod(ctx context.Context, orgId string, id string) (entities.SecurePaymentMethod, error)
    CreateSecurePaymentMethod(ctx context.Context, orgId string, input dto.CreatePaymentMethodInput) (entities.SecurePaymentMethod, error)
    UpdateSecurePaymentMethod(ctx context.Context, orgId string, paymentMethodId string, input dto.UpdatePaymentMethodInput) (entities.SecurePaymentMethod, error)
}
```

#### 2.3 Update Other Service Interfaces
Replace `request.Pagination` with `dto.Pagination` in:
- SubscriptionService
- PaymentService  
- OrderService
- DunningService
- InvoiceService

### Phase 3: Create Mapping Functions

Create mappers in `/internal/api/mappers/`:

#### 3.1 Customer Mappers (`customer_mappers.go`)
```go
package mappers

import (
    "payloop/internal/api/dto/request"
    "payloop/internal/api/dto/response"
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
)

// ToCreateCustomerInput converts API request to application input
func ToCreateCustomerInput(req request.CreateCustomerRequest) dto.CreateCustomerInput {
    return dto.CreateCustomerInput{
        Email:          req.Email,
        FirstName:      req.FirstName,
        LastName:       req.LastName,
        BillingAddress: req.BillingAddress,
        Phone:          req.Phone,
        Metadata:       req.Metadata,
    }
}

// ToCreatePaymentMethodInput converts API request to application input
func ToCreatePaymentMethodInput(customerId string, req request.CreatePaymentMethodRequest) dto.CreatePaymentMethodInput {
    return dto.CreatePaymentMethodInput{
        CustomerId:     customerId,
        Psp:            req.Psp,
        Name:           req.Name,
        Type:           req.Type,
        Details:        req.Details,
        Token:          req.Token,
        IsDefault:      req.IsDefault,
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
        Phone:          customer.Phone,
        CreatedAt:      customer.CreatedAt,
        UpdatedAt:      customer.UpdatedAt,
        Metadata:       customer.Metadata,
    }
}
```

#### 3.2 Usage Mappers (`usage_mappers.go`)
```go
package mappers

import (
    "payloop/internal/api/dto/request"
    "payloop/internal/api/dto/response"
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
)

// ToRecordUsageInput converts API request to application input
func ToRecordUsageInput(req request.RecordUsageRequest) dto.RecordUsageInput {
    return dto.RecordUsageInput{
        SubscriptionItemId: req.SubscriptionItemId,
        Quantity:          req.Quantity,
        TransactionValue:  req.TransactionValue,
        PercentageRate:    req.PercentageRate,
        ReferenceId:       req.ReferenceId,
        ReferenceType:     req.ReferenceType,
        Timestamp:         req.Timestamp,
        Metadata:          req.Metadata,
    }
}

// ToBatchRecordUsageInput converts API request to application input
func ToBatchRecordUsageInput(req request.BatchRecordUsageRequest) dto.BatchRecordUsageInput {
    records := make([]dto.RecordUsageInput, len(req.Records))
    for i, record := range req.Records {
        records[i] = ToRecordUsageInput(record)
    }
    return dto.BatchRecordUsageInput{
        Records: records,
    }
}

// ToUsageRecordResponse converts domain entity to API response
func ToUsageRecordResponse(record entities.UsageRecord) response.UsageRecordResponse {
    return response.UsageRecordResponse{
        Id:                 record.Id,
        SubscriptionId:     record.SubscriptionId,
        SubscriptionItemId: record.SubscriptionItemId,
        CustomerId:         record.CustomerId,
        PriceId:           record.PriceId,
        Quantity:          record.Quantity,
        Amount:            record.Amount,
        UsageDate:         record.UsageDate,
        BillingPeriod:     record.BillingPeriod,
        Processed:         record.Processed,
        ReferenceId:       record.ReferenceId,
        ReferenceType:     record.ReferenceType,
        CreatedAt:         record.CreatedAt,
        UpdatedAt:         record.UpdatedAt,
        Metadata:          record.Metadata,
    }
}

// ToUsageRecordListResponse converts paginated result to API response
func ToUsageRecordListResponse(result dto.PaginatedResult[entities.UsageRecord]) response.UsageRecordListResponse {
    items := make([]response.UsageRecordResponse, len(result.Items))
    for i, record := range result.Items {
        items[i] = ToUsageRecordResponse(record)
    }
    
    return response.UsageRecordListResponse{
        Items:      items,
        TotalCount: result.TotalCount,
        Page:       result.Page,
        PageSize:   result.PageSize,
        HasMore:    result.HasMore,
    }
}
```

#### 3.3 Common Mappers (`common_mappers.go`)
```go
package mappers

import (
    "payloop/internal/api/dto/request"
    "payloop/internal/application/dto"
)

// ToPagination converts API pagination to application pagination
func ToPagination(req request.Pagination) dto.Pagination {
    return dto.Pagination{
        Page:          req.Page,
        Limit:         req.Limit,
        Offset:        req.Offset,
        SortDirection: req.SortDirection,
        SortBy:        req.SortBy,
    }
}
```

### Phase 4: Update Service Implementations

#### 4.1 Update UsageRecordingService Implementation
- Replace all API DTO imports with application DTO imports
- Update method signatures to match new interface
- Remove response DTO creation logic (move to controllers)
- Use domain entities internally

#### 4.2 Update CustomerService Implementation
- Replace API DTO inputs with application DTO inputs
- Update method signatures
- Remove response DTO creation logic

#### 4.3 Update Other Service Implementations
- Replace pagination types
- Update method signatures

### Phase 5: Update Controllers

#### 5.1 Update Controllers to Use Mappers
Example for UsageController:
```go
// Before
func (c *UsageController) RecordUsage(ctx *gin.Context) {
    var req request.RecordUsageRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    result, err := c.usageService.RecordUsage(ctx, orgId, req)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    ctx.JSON(200, result)
}

// After
func (c *UsageController) RecordUsage(ctx *gin.Context) {
    var req request.RecordUsageRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Convert API DTO to application DTO
    input := mappers.ToRecordUsageInput(req)
    
    result, err := c.usageService.RecordUsage(ctx, orgId, input)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // Convert domain entity to API response
    response := mappers.ToUsageRecordResponse(result)
    ctx.JSON(200, response)
}
```

### Phase 6: Update Tests

#### 6.1 Service Tests
- Update test inputs to use application DTOs instead of API DTOs
- Verify domain entities are returned
- Test business logic without API concerns

#### 6.2 Integration Tests
- Test full request/response cycle through controllers
- Verify mapping logic works correctly

## Implementation Strategy

### Approach
1. **Incremental Implementation**: Implement one service at a time
2. **Backward Compatibility**: Maintain API compatibility during transition
3. **Test Coverage**: Ensure all changes are thoroughly tested
4. **Documentation**: Update documentation and examples

### Order of Implementation
1. Create all application DTOs first
2. Create mapper functions
3. Update service interfaces
4. Update service implementations
5. Update controllers to use mappers
6. Update tests
7. Remove unused API DTO dependencies from services

### Validation Criteria
- [ ] No `internal/api/dto/request` or `internal/api/dto/response` imports in application services
- [ ] All application services use domain entities or application DTOs
- [ ] API layer handles all conversion between API and application DTOs
- [ ] All tests pass
- [ ] No breaking changes to public API
- [ ] Code follows established DDD patterns

## Benefits

### Achieved After Implementation
1. **Clean Architecture Compliance**: Proper separation of concerns
2. **Improved Testability**: Application services can be tested in isolation
3. **Better Maintainability**: Changes to API don't affect business logic
4. **Consistent Patterns**: All services follow the same architectural patterns
5. **Domain Purity**: Business logic is not polluted with API concerns

### Technical Debt Reduction
- Eliminates architectural violations
- Establishes clear boundaries between layers
- Improves code organization and readability
- Makes future refactoring easier

## Risks and Mitigation

### Risks
1. **Breaking Changes**: Potential for introducing bugs during refactoring
2. **Large Scope**: Many files need to be changed
3. **Test Maintenance**: Many tests will need updates

### Mitigation
1. **Incremental Implementation**: Change one service at a time
2. **Comprehensive Testing**: Thorough testing at each step
3. **Code Review**: Careful review of all changes
4. **Feature Flags**: Use feature flags if needed for gradual rollout

## Success Metrics

### Completion Criteria
- [ ] All application services use only domain entities and application DTOs
- [ ] All API DTO conversions happen in the API layer
- [ ] All tests pass
- [ ] No architectural violations remain
- [ ] Documentation is updated

### Quality Metrics
- Zero imports of `internal/api/dto` in application layer
- All service methods return domain entities
- Clean separation between API and application layers
- Consistent patterns across all services