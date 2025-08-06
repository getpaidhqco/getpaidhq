# Public Token-Validated Payment Pages - Implementation Specification

## Overview
Create public endpoints that allow customers to view payment details and create orders using token-based authentication. All endpoints use query parameter tokens and are placed under `/api/pay/` routes.

## URL Structure
- **Base pattern**: `/api/pay/:slug`
- **Token format**: Query parameter `?token=abc123...`
- **Example**: `GET /api/pay/inv-2024-001?token=a7b9c3d4e5f6...`

## Required Endpoints

### 1. Get Payment Details
- **Route**: `GET /api/pay/:slug`
- **Purpose**: Retrieve invoice/checkout details for display
- **Token**: Required via query parameter
- **Returns**: Invoice data, payment configuration, organization info
- **Use Case**: Customer visits payment link and sees invoice details

### 2. Create Order
- **Route**: `POST /api/pay/:slug/create-order`
- **Purpose**: Create order and initiate payment with PSP
- **Token**: Required via query parameter
- **Input**: Payment processor, customer details, callback URLs
- **Returns**: Order ID, PSP-specific data (redirect URL, client secret, etc.)
- **Use Case**: Customer clicks "Pay Now" button

### 3. Get Order Status
- **Route**: `GET /api/pay/:slug/order/:orderId/status`
- **Purpose**: Check payment/order status
- **Token**: Required via query parameter
- **Returns**: Order status, payment status
- **Use Case**: Frontend polling for payment completion

## Authentication Changes
- Add `/api/pay/*` to PublicPaths in `internal/application/lib/authn/authn.go`
- No authentication middleware applied to these routes
- Token validation happens inside each controller method

## Token Validation Process
Each endpoint must perform these validation steps:
1. Extract token from query parameter `?token=`
2. Find PaymentLink by slug parameter
3. If PaymentLink has tokenHash field populated, validate token by hashing and comparing with stored hash
4. Check PaymentLink status is "active"
5. Check PaymentLink hasn't expired (if expiresAt is set)
6. For single-use links, check it hasn't been used already (usedAt field)

## Implementation Components

### Controller Structure
- **File**: `internal/api/controllers/public_payment_controller.go`
- **Name**: `PublicPaymentController` 
- Include shared token validation method that other methods can call
- Extract token from query parameters in each method
- Return appropriate HTTP status codes and error messages

### Routes Setup
- **File**: `internal/api/routes/public_payment.go`
- Set up route group `/api/pay`
- Register three endpoints listed above
- No middleware applied (routes are already marked as public)

### Request/Response DTOs
- **Request DTO**: For order creation with payment processor, customer details, callback URLs
- **Response DTOs**: For payment details and order creation responses
- Place in `internal/api/dto/request/` and `internal/api/dto/response/` folders

### Service Dependencies
- PublicPaymentController needs:
  - PaymentLinkService (to find by slug, mark as used)
  - OrderService (existing CreateOrder method)
  - InvoiceService (to get invoice details)

## Error Handling
- **401 Unauthorized**: Invalid token, missing token when required
- **404 Not Found**: Payment link not found, invalid slug
- **410 Gone**: Payment link expired
- **409 Conflict**: Single-use payment link already used
- **400 Bad Request**: Invalid request format
- **500 Internal Server Error**: System errors

Provide clear error messages for debugging without exposing system details.

## Integration Points

### Payment Link Creation
- Payment link creation should generate cryptographically secure tokens
- Store SHA256 hash of token in PaymentLink.tokenHash field
- Include token in URLs sent to customers

### Order Creation
- Use existing OrderService.CreateOrder method
- Convert payment link data to order creation input
- Support both invoice-based and direct checkout payment links

### PSP Integration
- PSP integration remains unchanged
- Return same data structure as existing initiate payment endpoint
- Support redirect URLs, client secrets, session IDs based on PSP type

## Frontend Integration

### Token Extraction
- Frontend extracts token from URL query parameters
- All API calls include token as query parameter
- SSR frameworks can access query parameters during server-side rendering
- Client-side JavaScript can extract from window.location.search

### API Call Pattern
```
// Get payment details
GET /api/pay/inv-2024-001?token=abc123

// Create order
POST /api/pay/inv-2024-001/create-order?token=abc123

// Check status
GET /api/pay/inv-2024-001/order/ORD-456/status?token=abc123
```

## Files to Create/Modify

### New Files
1. `internal/api/controllers/public_payment_controller.go`
2. `internal/api/routes/public_payment.go`
3. `internal/api/dto/request/public_payment.go`
4. `internal/api/dto/response/public_payment.go`

### Modified Files
1. `internal/application/lib/authn/authn.go` - add public paths
2. `internal/application/services/payment_link_service.go` - add FindBySlug method
3. Bootstrap/modules files - register new routes and controller

## PaymentLink Data Structure

### Required Fields
- `slug`: Unique identifier for the payment link
- `tokenHash`: SHA256 hash of access token (nullable for public links)
- `data`: JSON containing payment context (invoice ID, checkout items, etc.)
- `config`: JSON containing payment configuration (allowed methods, UI options)
- `status`: PaymentLinkStatus enum (active, used, expired, disabled)
- `singleUse`: Boolean flag for one-time use links
- `expiresAt`: Optional expiration timestamp
- `usedAt`: Timestamp when link was used (for single-use links)

### Data Field Structure
```json
{
  "type": "invoice",
  "invoiceId": "INV-123"
}
```

Or for direct checkout:
```json
{
  "type": "checkout",
  "items": [...]
}
```

## Security Considerations

### Token Security
- Tokens must be cryptographically secure (minimum 32 bytes entropy)
- Store only hashed versions in database
- Tokens are single-use for single-use payment links
- Tokens expire with payment link expiration

### Access Control
- Token validates access to specific payment link only
- No cross-payment-link access allowed
- Tenant isolation maintained through PaymentLink.orgId

### Error Information
- Don't leak information about valid slugs in error messages
- Don't expose token validation details
- Generic error messages for security

## Success Criteria
1. Customer can view invoice details using payment link URL with token
2. Customer can create order and get PSP payment data without authentication
3. Token validation prevents unauthorized access to payment links
4. Single-use links are properly enforced and marked as used
5. Frontend can make all necessary API calls without database access
6. Existing authenticated endpoints remain unchanged
7. Multi-tenant security is maintained
8. Error handling provides appropriate feedback without security leaks

## Testing Requirements
1. Test token validation with valid, invalid, and missing tokens
2. Test payment link status validation (active, used, expired, disabled)
3. Test single-use link enforcement
4. Test different payment link types (invoice, checkout)
5. Test error scenarios and appropriate status codes
6. Test tenant isolation
7. Test token extraction from query parameters
8. Integration tests with frontend token passing