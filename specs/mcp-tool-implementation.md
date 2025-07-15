# MCP Tool Suite Implementation Specification

## Overview

This specification defines the implementation of a comprehensive Model Context Protocol (MCP) tool suite for PayLoop, enabling AI assistants to perform complete subscription billing operations. The implementation builds upon the existing MCP server foundation at `internal/mcp/` and expands it to cover all business operations.

## Current State Analysis

### Existing Implementation
- **MCP Server**: Fully implemented with SSE protocol support (`internal/mcp/server.go`)
- **Dependency Injection**: Integrated with Uber FX (`internal/mcp/fx.go`)
- **Module Registration**: Active in application bootstrap (`internal/application/bootstrap/modules.go:31`)
- **Server Startup**: Runs on configurable port alongside main HTTP server
- **Existing Tools**: 
  - `hello_world` - Basic demonstration tool (complete)
  - `create_invoice` - Invoice creation tool (incomplete - returns mock data)

### Available Services
All required application services are available through dependency injection:
- `CustomerService`, `SubscriptionOrchestrationService`, `ProductService`
- `InvoiceService`, `PaymentService`, `UsageRecordingService`
- `ReportService`, `DunningOrchestrationService`, `OrderService`
- `CartService`, `SessionService`, `SettingsService`

## Architecture Requirements

### 1. Clean Architecture Compliance
All MCP tools MUST follow PayLoop's clean architecture patterns:

```go
// ✅ CORRECT: Tools use application services with authenticated context
func CustomerHandler(ctx context.Context, request mcp.CallToolRequest, 
                    customerService interfaces.CustomerService, 
                    logger logger.Logger, authCtx *AuthContext) (*mcp.CallToolResult, error) {
    
    // Org ID comes from authenticated user context
    orgId := authCtx.OrgId
    
    // Extract parameters from MCP request (no org_id needed)
    email, err := request.RequireString("email")
    firstName, err := request.RequireString("first_name")
    
    // Create application DTO (NOT API DTO)
    input := dto.CreateCustomerInput{
        Email:     email,
        FirstName: firstName,
        // ... other fields
    }
    
    // Call application service with authenticated org_id
    customer, err := customerService.Create(ctx, orgId, input)
    
    // Return domain entity data
    return mcp.NewToolResultText(fmt.Sprintf("Customer created: %s", customer.Id)), nil
}

// ❌ WRONG: Never use API DTOs in MCP handlers
import "payloop/internal/api/dto/request" // FORBIDDEN
```

### 2. Authentication & Multi-Tenancy Requirements
All MCP tools MUST be authenticated and automatically enforce organization isolation:

```go
// Authentication is handled by middleware - org_id extracted from auth tokens
// Tools receive AuthContext with authenticated user and org_id
func ToolHandler(ctx context.Context, request mcp.CallToolRequest, 
                service interfaces.Service, logger logger.Logger,
                authCtx *AuthContext) (*mcp.CallToolResult, error) {
    
    // Org ID comes from authenticated user - no manual parameter needed
    orgId := authCtx.OrgId
    userId := authCtx.User.Id
    
    // Log authenticated operation
    logger.Info("MCP operation", 
        "tool", "tool_name",
        "orgId", orgId, 
        "userId", userId)
    
    // Pass orgId as first parameter to all service calls
    result, err := service.Operation(ctx, orgId, parameters...)
}
```

#### Authentication Methods Supported
- **API Keys**: `Authorization: Bearer pk_live_...` or `Authorization: Bearer pk_test_...`
- **Clerk OAuth**: `Authorization: Bearer clerk_session_token`
- **Both methods**: Automatically extract `orgId` from validated tokens

### 3. Error Handling Standards
Consistent error handling across all tools:

```go
// Service errors should be logged and returned as tool errors
if err != nil {
    logger.Error("Operation failed", 
        "operation", "create_customer",
        "orgId", orgId,
        "error", err.Error())
    return mcp.NewToolResultError(fmt.Sprintf("Operation failed: %s", err.Error())), nil
}

// Success responses should be informative
return mcp.NewToolResultText(fmt.Sprintf("Operation completed successfully. ID: %s", result.Id)), nil
```

### 4. Parameter Validation
All tools must validate required and optional parameters:

```go
// Required parameters
customerId, err := request.RequireString("customer_id")
if err != nil {
    return mcp.NewToolResultError("customer_id is required"), nil
}

// Optional parameters with defaults
limit, err := request.RequireInt("limit")
if err != nil || limit <= 0 {
    limit = 20 // Default value
}

// Enum validation
status, err := request.RequireString("status")
if err == nil && !isValidStatus(status) {
    return mcp.NewToolResultError("Invalid status. Valid values: active, inactive, cancelled"), nil
}
```

## Implementation Phases

### Phase 1: Core Business Operations (Priority: CRITICAL)
**Timeline: Weeks 1-3**
**Acceptance Criteria: AI can handle basic subscription operations**

#### 1.1 Customer Management Tools
**Files to Create:**
- `internal/mcp/tools/customer.go` - Tool definitions
- `internal/mcp/handlers/customer.go` - Handler implementations

**Tools to Implement:**
```go
// Customer CRUD (org_id automatically extracted from auth)
create_customer(email, first_name, last_name, billing_address?, metadata?)
get_customer(customer_id)
list_customers(page?, limit?, status?, email_filter?)
update_customer(customer_id, first_name?, last_name?, billing_address?, metadata?)

// Customer Payment Methods  
add_payment_method(customer_id, payment_method_data)
update_payment_method(customer_id, payment_method_id, payment_method_data)
get_payment_method(payment_method_id)

// Customer Relations
list_customer_invoices(customer_id, page?, limit?, status?)
get_customer_dunning_history(customer_id)
```

**Service Dependencies:**
- `CustomerService.Create()`, `CustomerService.FindById()`, `CustomerService.List()`
- `CustomerService.CreatePaymentMethod()`, `CustomerService.UpdatePaymentMethod()`

#### 1.2 Subscription Lifecycle Tools
**Files to Create:**
- `internal/mcp/tools/subscription.go`
- `internal/mcp/handlers/subscription.go`

**Tools to Implement:**
```go
// Subscription CRUD (org_id from auth context)
get_subscription(subscription_id)
list_subscriptions(page?, limit?, status?, customer_id?, sort_by?, sort_direction?)
update_subscription(subscription_id, metadata?)

// Subscription State Management
pause_subscription(subscription_id, pause_reason?)
resume_subscription(subscription_id)
cancel_subscription(subscription_id, cancel_reason?, immediate?)
change_plan(subscription_id, new_price_id, proration?)
update_billing_anchor(subscription_id, billing_anchor_date)

// Subscription Relations
list_subscription_payments(subscription_id, page?, limit?)
get_subscription_usage(subscription_id, start_date?, end_date?)
```

**Service Dependencies:**
- `SubscriptionOrchestrationService.Pause()`, `SubscriptionOrchestrationService.Resume()`
- `SubscriptionOrchestrationService.Cancel()`, `SubscriptionOrchestrationService.ChangePlan()`

#### 1.3 Product & Pricing Tools
**Files to Create:**
- `internal/mcp/tools/product.go`
- `internal/mcp/handlers/product.go`

**Tools to Implement:**
```go
// Product Management (org_id from auth context)
create_product(name, description?, metadata?)
get_product(product_id)
list_products(page?, limit?)
update_product(product_id, name?, description?, metadata?)
delete_product(product_id)

// Product Variants
create_variant(product_id, name, description?, metadata?)
get_variant(variant_id)
list_variants(product_id, page?, limit?)
update_variant(variant_id, name?, description?, metadata?)
delete_variant(variant_id)

// Pricing (with full usage-based billing support)
create_price(variant_id, category, currency, unit_price?, scheme?, billing_interval?, 
            has_usage?, usage_type?, aggregation_type?, percentage_rate?, fixed_fee?, 
            overage_unit_price?, included_usage?, usage_limit?, tiers?, metadata?)
get_price(price_id)
list_prices(variant_id, page?, limit?)
update_price(price_id, [same parameters as create_price])
delete_price(price_id)
```

**Service Dependencies:**
- `ProductService.CreateProduct()`, `ProductService.FindById()`, `ProductService.List()`
- `ProductService.CreateVariant()`, `ProductService.GetVariant()`, `ProductService.ListVariants()`
- `ProductService.CreateProductPrice()`, `ProductService.GetPrice()`, `ProductService.ListPrices()`

### Phase 2: Revenue Operations (Priority: HIGH)
**Timeline: Weeks 3-5**
**Acceptance Criteria: AI can handle billing and payment operations**

#### 2.1 Complete Invoice Management
**Files to Modify:**
- `internal/mcp/handlers/invoice.go` - Complete existing implementation

**Current Issue:**
```go
// Line 85-91 in handlers/invoice.go - REPLACE MOCK IMPLEMENTATION
// In a real implementation, you would call the invoice service to create the invoice
// For now, we'll just return a success message
// Use a mock invoice ID
invoiceId := "inv_123456789"
```

**Required Implementation:**
```go
// Replace mock with actual service call
invoice, err := invoiceService.Create(ctx, orgId, dto.CreateInvoiceInput{
    CustomerId:    customerId,
    Type:         docType,
    InvoiceType:  invoiceType,
    Currency:     currency,
    OrderId:      orderId,
    SubscriptionId: subscriptionId,
    DueAt:        dueAt,
    Notes:        notes,
    CustomerNotes: customerNotes,
})

if err != nil {
    logger.Error("Failed to create invoice", "error", err.Error())
    return mcp.NewToolResultError(fmt.Sprintf("Failed to create invoice: %s", err.Error())), nil
}

return mcp.NewToolResultText(fmt.Sprintf("Invoice created successfully with ID: %s", invoice.Id)), nil
```

**Additional Invoice Tools:**
```go
// Invoice CRUD
get_invoice(org_id, invoice_id)
list_invoices(org_id, page?, limit?, status?, customer_id?)
update_invoice(org_id, invoice_id, notes?, customer_notes?, due_at?)

// Invoice Line Items
add_line_item(org_id, invoice_id, description, quantity, unit_price, metadata?)
update_line_item(org_id, invoice_id, line_item_id, description?, quantity?, unit_price?, metadata?)
delete_line_item(org_id, invoice_id, line_item_id)
list_line_items(org_id, invoice_id)

// Invoice Operations
generate_invoice_pdf(org_id, invoice_id, template?)
get_invoice_history(org_id, invoice_id)
perform_invoice_action(org_id, invoice_id, action) // send, mark_paid, void, etc.
```

#### 2.2 Payment Management Tools
**Files to Create:**
- `internal/mcp/tools/payment.go`
- `internal/mcp/handlers/payment.go`

**Tools to Implement:**
```go
// Payment Operations
get_payment(org_id, payment_id)
list_payments(org_id, page?, limit?, status?, customer_id?)
refund_payment(org_id, payment_id, amount?, reason?)

// Payment Method Management
get_payment_method(org_id, payment_method_id)
```

#### 2.3 Order Processing Tools
**Files to Create:**
- `internal/mcp/tools/order.go`
- `internal/mcp/handlers/order.go`

**Tools to Implement:**
```go
// Order Management
create_order(org_id, customer_id, line_items, metadata?)
get_order(org_id, order_id)
list_orders(org_id, page?, limit?, status?, customer_id?)
complete_order(org_id, order_id, payment_method_id?)
list_order_subscriptions(org_id, order_id)

// Cart Operations
add_to_cart(org_id, cart_id, product_variant_id, quantity, metadata?)
remove_from_cart(org_id, cart_id, item_id)

// Session Management
create_session(org_id, customer_id, success_url, cancel_url, metadata?)
```

### Phase 3: Usage-Based Billing & Analytics (Priority: MEDIUM)
**Timeline: Weeks 5-7**
**Acceptance Criteria: AI can handle modern SaaS billing and provide business insights**

#### 3.1 Usage Recording & Metering
**Files to Create:**
- `internal/mcp/tools/usage.go`
- `internal/mcp/handlers/usage.go`

**Tools to Implement:**
```go
// Usage Recording
record_usage(org_id, subscription_item_id, usage_type, quantity, timestamp?, metadata?)
batch_record_usage(org_id, usage_records[])
get_usage_record(org_id, usage_record_id)
list_usage_records(org_id, subscription_item_id, page?, limit?, start_date?, end_date?)
delete_usage_record(org_id, usage_record_id)

// Usage Analytics
get_usage_summary(org_id, subscription_item_id, start_date?, end_date?, aggregation_type?)
get_subscription_usage(org_id, subscription_id, start_date?, end_date?)
```

#### 3.2 Revenue Analytics Tools
**Files to Create:**
- `internal/mcp/tools/analytics.go`
- `internal/mcp/handlers/analytics.go`

**Tools to Implement:**
```go
// Revenue Metrics
get_mrr(org_id, start_date?, end_date?, granularity?)
get_arr(org_id, start_date?, end_date?)
get_active_subscribers(org_id, date?)
get_refund_totals(org_id, start_date?, end_date?, granularity?)

// Churn Analysis
get_churn_totals(org_id, start_date?, end_date?, granularity?)
get_churn_rates(org_id, start_date?, end_date?, granularity?)
```

### Phase 4: Customer Success & Recovery (Priority: MEDIUM)
**Timeline: Weeks 7-8**
**Acceptance Criteria: AI can manage failed payment recovery and customer retention**

#### 4.1 Dunning Management Tools
**Files to Create:**
- `internal/mcp/tools/dunning.go`
- `internal/mcp/handlers/dunning.go`

**Tools to Implement:**
```go
// Dunning Campaigns
list_dunning_campaigns(org_id, page?, limit?, status?)
get_dunning_campaign(org_id, campaign_id)
update_dunning_campaign(org_id, campaign_id, metadata?)

// Dunning Attempts
list_dunning_attempts(org_id, campaign_id, page?, limit?)
trigger_manual_attempt(org_id, campaign_id, subscription_id, attempt_type?)
list_dunning_communications(org_id, campaign_id, page?, limit?)

// Dunning Configuration
create_dunning_config(org_id, name, retry_schedule, communication_templates, metadata?)
get_dunning_config(org_id, config_id)
list_dunning_configs(org_id, page?, limit?)
update_dunning_config(org_id, config_id, name?, retry_schedule?, communication_templates?, metadata?)

// Payment Recovery
create_payment_token(org_id, subscription_id, customer_id)
verify_payment_token(org_id, token)
activate_payment_token(org_id, token, payment_method_data)
```

### Phase 5: Advanced AI Features (Priority: LOW)
**Timeline: Weeks 9-10**
**Acceptance Criteria: AI can provide intelligent insights and automation**

#### 5.1 Intelligent Analysis Tools
**Files to Create:**
- `internal/mcp/tools/intelligence.go`
- `internal/mcp/handlers/intelligence.go`

**Tools to Implement:**
```go
// Customer Intelligence
analyze_customer_health(org_id, customer_id)
predict_churn_risk(org_id, customer_id?, time_horizon?)
recommend_plan_change(org_id, subscription_id)
analyze_usage_patterns(org_id, subscription_id, time_period?)

// Revenue Intelligence
generate_revenue_forecast(org_id, forecast_period, confidence_level?)
identify_expansion_opportunities(org_id, customer_id?, revenue_threshold?)
```

#### 5.2 Automated Workflow Tools
**Files to Create:**
- `internal/mcp/tools/automation.go`
- `internal/mcp/handlers/automation.go`

**Tools to Implement:**
```go
// Bulk Operations
bulk_update_subscriptions(org_id, subscription_ids[], update_data, confirmation_required?)
bulk_apply_discounts(org_id, customer_ids[], discount_data, expiration_date?)

// Automated Reports
generate_customer_report(org_id, customer_id, report_type, format?)
schedule_dunning_campaign(org_id, config_id, trigger_conditions, schedule?)

// External Integration
sync_external_usage(org_id, external_system, sync_config, dry_run?)
```

### Phase 6: Configuration & Management (Priority: LOW)
**Timeline: Week 10**
**Acceptance Criteria: AI can manage system configuration**

#### 6.1 Settings & Configuration Tools
**Files to Create:**
- `internal/mcp/tools/settings.go`
- `internal/mcp/handlers/settings.go`

**Tools to Implement:**
```go
// Settings Management
get_settings(org_id, parent_id?, setting_key?)
list_settings(org_id, parent_id?, page?, limit?)
create_setting(org_id, parent_id, key, value, metadata?)
update_setting(org_id, parent_id, setting_id, value?, metadata?)
delete_setting(org_id, parent_id, setting_id)

// Webhook Management
create_webhook_subscription(org_id, endpoint_url, events[], secret?, metadata?)
list_webhooks(org_id, page?, limit?)
create_webhook(org_id, config_data)

// System Health
health_check()
```

## Authentication Implementation

### 1. Authentication Middleware
Create authentication middleware to extract org_id from tokens:

**File: `internal/mcp/middleware/auth.go`**
```go
package middleware

import (
    "context"
    "errors"
    "strings"
    "payloop/internal/api/authn"
    "payloop/internal/infrastructure/authn/apikey"
    "payloop/internal/infrastructure/authn/clerk"
)

type AuthContext struct {
    User   authn.User
    OrgId  string
}

func ExtractAuthFromMCPRequest(ctx context.Context, headers map[string]string) (*AuthContext, error) {
    authHeader := headers["Authorization"]
    if authHeader == "" {
        return nil, errors.New("authentication required: Authorization header missing")
    }
    
    // Try API key authentication first
    if strings.HasPrefix(authHeader, "Bearer pk_live_") || strings.HasPrefix(authHeader, "Bearer pk_test_") {
        user, err := apikey.ValidateAPIKey(ctx, authHeader)
        if err == nil {
            return &AuthContext{
                User:  user,
                OrgId: user.OrgId,
            }, nil
        }
    }
    
    // Try Clerk OAuth authentication
    if strings.HasPrefix(authHeader, "Bearer clerk_") {
        user, err := clerk.ValidateSessionToken(ctx, authHeader)
        if err == nil {
            return &AuthContext{
                User:  user,
                OrgId: user.OrgId,
            }, nil
        }
    }
    
    return nil, errors.New("invalid authentication token")
}
```

### 2. MCP Server Authentication Integration
Update MCP server to handle authentication:

**File: `internal/mcp/server.go`**
```go
func (s *MCPServer) handleToolCall(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract authentication from request headers
    authCtx, err := middleware.ExtractAuthFromMCPRequest(ctx, request.Headers)
    if err != nil {
        s.logger.Warn("MCP authentication failed", "error", err.Error())
        return mcp.NewToolResultError("Authentication required"), nil
    }
    
    // Log authenticated operation
    s.logger.Info("MCP tool call", 
        "tool", request.Name,
        "orgId", authCtx.OrgId,
        "userId", authCtx.User.Id,
        "userEmail", authCtx.User.Email)
    
    // Route to appropriate handler with auth context
    return s.routeToHandler(ctx, request, authCtx)
}
```

### 3. Client Usage Examples

#### Using API Key Authentication
```javascript
// MCP Client example with API key
const mcpClient = new MCPClient({
  url: "http://localhost:8084",
  headers: {
    "Authorization": "Bearer pk_live_1234567890abcdef..."
  }
});

// Create customer (no org_id needed)
const result = await mcpClient.callTool("create_customer", {
  email: "customer@example.com",
  first_name: "John", 
  last_name: "Doe"
});
```

#### Using Clerk Session Token
```javascript
// MCP Client example with Clerk OAuth
const mcpClient = new MCPClient({
  url: "http://localhost:8084", 
  headers: {
    "Authorization": "Bearer clerk_session_token_here"
  }
});

// List subscriptions (org_id automatically scoped)
const subscriptions = await mcpClient.callTool("list_subscriptions", {
  status: "active",
  limit: 50
});
```

## Technical Implementation Guidelines

### 1. File Structure Standards
All MCP tools must follow this structure:

```
internal/mcp/
├── server.go           # Main server with tool registration
├── fx.go              # Dependency injection
├── tools/             # Tool definitions
│   ├── customer.go    # Customer tool definitions
│   ├── subscription.go
│   ├── product.go
│   ├── invoice.go
│   ├── payment.go
│   ├── usage.go
│   ├── analytics.go
│   ├── dunning.go
│   ├── intelligence.go
│   ├── automation.go
│   └── settings.go
└── handlers/          # Handler implementations
    ├── customer.go    # Customer handler implementations
    ├── subscription.go
    ├── product.go
    ├── invoice.go     # Complete existing implementation
    ├── payment.go
    ├── usage.go
    ├── analytics.go
    ├── dunning.go
    ├── intelligence.go
    ├── automation.go
    └── settings.go
```

### 2. Tool Definition Pattern
Every tool file must follow this pattern:

```go
package tools

import "github.com/mark3labs/mcp-go/mcp"

// NewCreateCustomerTool creates a new customer creation tool
func NewCreateCustomerTool() mcp.Tool {
    return mcp.NewTool("create_customer",
        mcp.WithDescription("Create a new customer account (org_id automatically extracted from auth)"),
        mcp.WithString("email",
            mcp.Required(),
            mcp.Description("Customer email address (must be unique)"),
        ),
        mcp.WithString("first_name",
            mcp.Required(),
            mcp.Description("Customer first name"),
        ),
        mcp.WithString("last_name",
            mcp.Required(),
            mcp.Description("Customer last name"),
        ),
        mcp.WithObject("billing_address",
            mcp.Description("Customer billing address (optional)"),
        ),
        mcp.WithObject("metadata",
            mcp.Description("Additional metadata as key-value pairs (optional)"),
        ),
    ) 
}
```

The tool definition must match the handler implementation, ensuring that all required parameters are defined and documented.


### 3. Handler Implementation Pattern
Every handler file must follow this pattern:

```go
package handlers

import (
    "context"
    "fmt"
    "payloop/internal/application/interfaces"
    "payloop/internal/application/lib/logger"
    "payloop/internal/application/dto"
    
    "github.com/mark3labs/mcp-go/mcp"
)

// CreateCustomerHandler handles customer creation requests
func CreateCustomerHandler(ctx context.Context, request mcp.CallToolRequest, 
                          customerService interfaces.CustomerService, 
                          logger logger.Logger, authCtx *AuthContext) (*mcp.CallToolResult, error) {
    
    // 1. Get org_id from authenticated context (no manual parameter needed)
    orgId := authCtx.OrgId
    userId := authCtx.User.Id
    
    // 2. Extract and validate required parameters (no org_id needed)
    email, err := request.RequireString("email")
    if err != nil {
        return mcp.NewToolResultError("email is required"), nil
    }
    
    firstName, err := request.RequireString("first_name")
    if err != nil {
        return mcp.NewToolResultError("first_name is required"), nil
    }
    
    lastName, err := request.RequireString("last_name")
    if err != nil {
        return mcp.NewToolResultError("last_name is required"), nil
    }
    
    // 3. Extract optional parameters
    var billingAddress *entities.Address
    if addr, err := request.GetObject("billing_address"); err == nil && addr != nil {
        // Parse billing address from map[string]interface{}
        billingAddress = parseBillingAddress(addr)
    }
    
    var metadata map[string]string
    if meta, err := request.GetObject("metadata"); err == nil && meta != nil {
        metadata = parseMetadata(meta)
    }
    
    // 4. Create application DTO (NOT API DTO)
    input := dto.CreateCustomerInput{
        Email:          email,
        FirstName:      firstName,
        LastName:       lastName,
        BillingAddress: billingAddress,
        Metadata:       metadata,
    }
    
    // 5. Log authenticated operation
    logger.Info("Creating customer via MCP",
        "orgId", orgId,
        "userId", userId,
        "email", email,
        "operation", "create_customer")
    
    // 6. Call application service with authenticated org_id
    customer, err := customerService.Create(ctx, orgId, input)
    if err != nil {
        logger.Error("Failed to create customer",
            "orgId", orgId,
            "userId", userId,
            "email", email,
            "error", err.Error())
        return mcp.NewToolResultError(fmt.Sprintf("Failed to create customer: %s", err.Error())), nil
    }
    
    // 7. Return success result
    return mcp.NewToolResultText(fmt.Sprintf("Customer created successfully. ID: %s, Email: %s", 
        customer.Id, customer.Email)), nil
}
```

### 4. Server Registration Pattern
All tools must be registered in `server.go`:

```go
func NewServer(params NewServerParams) MCPServer {
    s := server.NewMCPServer("getpaidhq-mcp 🚀", "1.0.0")
    
    // Customer tools
    s.AddTool(tools.NewCreateCustomerTool(), 
        func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            return handlers.CreateCustomerHandler(ctx, request, params.CustomerService, params.Logger)
        })
    
    s.AddTool(tools.NewGetCustomerTool(),
        func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            return handlers.GetCustomerHandler(ctx, request, params.CustomerService, params.Logger)
        })
    
    // Continue for all tools...
    
    return MCPServer{
        SSEServer:      server.NewSSEServer(s, server.WithBaseURL(":"+params.Env.McpSsePort)),
        logger:         params.Logger,
        // Store all services for handler access
    }
}
```

### 5. Service Dependency Management
Update `fx.go` to include all required services:

```go
type NewServerParams struct {
    fx.In

    Logger                     logger.Logger
    Env                       lib.Env
    CustomerService           interfaces.CustomerService
    SubscriptionService       interfaces.SubscriptionOrchestrationService
    ProductService            interfaces.ProductService
    InvoiceService            interfaces.InvoiceService
    PaymentService            interfaces.PaymentService
    UsageRecordingService     interfaces.UsageRecordingService
    ReportService             interfaces.ReportService
    DunningService            interfaces.DunningOrchestrationService
    OrderService              interfaces.OrderService
    CartService               interfaces.CartService
    SessionService            interfaces.SessionService
    SettingsService           interfaces.SettingsService
    WebhookSubscriptionService interfaces.WebhookSubscriptionService
}
```

## Quality Assurance Requirements

### 1. Testing Requirements
Each phase must include comprehensive testing:

```go
// Create test files alongside handler files
internal/mcp/handlers/customer_test.go
internal/mcp/handlers/subscription_test.go
// etc.

// Test structure example
func TestCreateCustomerHandler(t *testing.T) {
    tests := []struct {
        name           string
        request        map[string]interface{}
        mockResponse   entities.Customer
        mockError      error
        expectedResult string
        expectedError  string
    }{
        {
            name: "successful customer creation",
            request: map[string]interface{}{
                "org_id": "org_123",
                "email": "test@example.com",
                "first_name": "John",
                "last_name": "Doe",
            },
            mockResponse: entities.Customer{
                Id: "cust_123",
                Email: "test@example.com",
            },
            expectedResult: "Customer created successfully. ID: cust_123, Email: test@example.com",
        },
        // Add error cases, validation cases, etc.
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 2. Performance Requirements
- Each tool handler must complete within 30 seconds
- Bulk operations must support pagination to prevent timeouts
- All database queries must use proper indexing (already implemented in services)

### 3. Security Requirements
- All tools must validate `org_id` parameter for multi-tenancy
- No tools should bypass authentication/authorization patterns
- Sensitive data (payment methods, tokens) must be handled securely
- All user inputs must be validated and sanitized

### 4. Documentation Requirements
Each tool must be documented with:
- Purpose and use cases
- Required and optional parameters
- Expected return values
- Error conditions
- Usage examples

## Acceptance Criteria

### Phase 1 Acceptance Criteria
- [ ] AI can create, retrieve, and list customers
- [ ] AI can manage customer payment methods
- [ ] AI can retrieve and modify subscriptions
- [ ] AI can pause, resume, and cancel subscriptions
- [ ] AI can create and manage products, variants, and prices
- [ ] All tools enforce multi-tenancy with `org_id`
- [ ] All tools return meaningful success/error messages
- [ ] Complete test coverage for all implemented tools

### Phase 2 Acceptance Criteria
- [ ] Existing `create_invoice` tool returns actual invoice data (not mock)
- [ ] AI can manage complete invoice lifecycle
- [ ] AI can process payments and refunds
- [ ] AI can manage orders and cart operations
- [ ] AI can create checkout sessions
- [ ] All revenue operations work end-to-end

### Phase 3 Acceptance Criteria
- [ ] AI can record and retrieve usage data
- [ ] AI can generate usage summaries and reports
- [ ] AI can access MRR, ARR, and churn analytics
- [ ] All analytics tools support date range filtering

### Phase 4 Acceptance Criteria
- [ ] AI can manage dunning campaigns and attempts
- [ ] AI can configure dunning strategies
- [ ] AI can generate payment recovery tokens
- [ ] Customer success workflows are fully functional

### Phase 5 Acceptance Criteria
- [ ] AI can provide intelligent customer insights
- [ ] AI can perform bulk operations safely
- [ ] AI can generate automated reports
- [ ] Advanced analytics provide actionable insights

### Phase 6 Acceptance Criteria
- [ ] AI can manage system settings and configuration
- [ ] AI can monitor system health
- [ ] AI can configure webhooks and integrations
- [ ] All configuration changes are properly audited

## Deployment & Configuration

### Environment Variables
No new environment variables required. Uses existing:
- `GETPAIDHQ_MCP_SSE_PORT` - MCP server port (default: 8084)

### Server Startup
MCP server starts automatically with main application:
```bash
go run main.go serve
# MCP server accessible at http://localhost:8084
```

### Client Connection
AI assistants can connect via:
- **Protocol**: SSE (Server-Sent Events)
- **URL**: `http://localhost:8084`
- **Authentication**: Currently none (consider adding for production)

## Success Metrics

### Technical Metrics
- **Tool Count**: 60+ tools implemented across 15 categories
- **Coverage**: 100% of API endpoints accessible via MCP
- **Performance**: All tools respond within 30 seconds
- **Reliability**: 99.9% uptime for MCP server

### Business Metrics
- **AI Capability**: Complete subscription billing operations
- **Automation**: 80% reduction in manual billing tasks
- **Customer Support**: AI can resolve 90% of billing inquiries
- **Revenue Operations**: AI can handle end-to-end billing workflows

### Quality Metrics
- **Test Coverage**: 90%+ code coverage for all handlers
- **Documentation**: 100% of tools documented with examples
- **Error Handling**: Graceful degradation for all failure scenarios
- **Security**: Zero security vulnerabilities in multi-tenant isolation

## Conclusion

This specification provides a comprehensive roadmap for implementing a complete MCP tool suite that transforms PayLoop into a fully AI-accessible subscription billing platform. The phased approach ensures critical functionality is delivered first, while the detailed technical requirements ensure maintainable, secure, and scalable implementation.

The resulting system will enable AI assistants to perform any billing operation a human administrator could do, from basic customer management to complex revenue analytics and automated workflows.