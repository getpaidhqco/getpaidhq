# Discount System Design

## Overview
This document outlines the design for implementing a discount system in Payloop, following DDD principles and clean architecture patterns.

## Core Concepts

### Discount Types
- **Fixed Amount**: Deducts a specific monetary amount (e.g., $10 off), stored in smallest currency unit
- **Percentage**: Deducts a percentage of the total (e.g., 20% off), stored as 0-100

### Discount Application
- **Once**: Applied only to the first billing cycle
- **Forever**: Applied to all future billing cycles
- **Cycles**: Applied for a specified number of billing cycles

### Key Features
1. **Case-insensitive discount names** for easy management
2. **Optional discount codes** that are unique per organization
3. **Time-based validity** with optional start and end dates
4. **Redemption limits** to control usage
5. **Multi-currency support** for fixed amount discounts
6. **Flexible redemption tracking** linked to various resource types
7. **Dynamic redemption counting** to ensure data consistency (no denormalized counts)

## Architecture Overview

### Domain Layer
- **Discount Entity**: Core business logic for discount rules and validation
- **DiscountRedemption Entity**: Tracks usage and links to applied resources
- **Repository Interfaces**: Clean contracts for data persistence

### Application Layer
- **DiscountService**: Business operations and orchestration
- **DTOs**: Application-specific data structures
- **Event Publishing**: Discount lifecycle events

### API Layer
- **Controllers**: HTTP endpoints for discount management
- **Request/Response DTOs**: API-specific data structures with validation
- **Mappers**: Convert between API and Application DTOs

### Infrastructure Layer
- **PostgreSQL Repositories**: Implement domain repository interfaces
- **Event Publishers**:  integration for event messaging

## Database Schema

```prisma
// Add to prisma/schema.prisma

model Discount {
  id                  String    @id @default(cuid())
  orgId               String    @map("org_id")
  name                String
  type                String    // "fixed" or "percentage"
  value               Int       // For fixed: amount in smallest currency unit. For percentage: 0-100 (e.g., 20 = 20%)
  code                String?   
  startsAt            DateTime? @map("starts_at")
  endsAt              DateTime? @map("ends_at")
  maxRedemptions      Int?      @map("max_redemptions")
  recurring           String    // "once", "forever", "cycles"
  cycles              Int?      // Number of billing cycles when recurring is "cycles"
  currency            String?   // Required for fixed discounts
  active              Boolean   @default(true)
  createdAt           DateTime  @default(now()) @map("created_at")
  updatedAt           DateTime  @updatedAt @map("updated_at")
  metadata            Json?
  
  redemptions         DiscountRedemption[]
  
  @@unique([orgId, code])
  @@index([orgId, active])
  @@index([orgId, code])
  @@map("discounts")
}

model DiscountRedemption {
  id             String   @id @default(cuid())
  orgId          String   @map("org_id")
  discountId     String   @map("discount_id")
  customerId     String   @map("customer_id")
  resourceType   String   @map("resource_type") // "subscription", "invoice", "payment", "checkout_session"
  resourceId     String   @map("resource_id")
  discountAmount Int      @map("discount_amount") // Amount saved in smallest currency unit
  currency       String
  createdAt      DateTime @default(now()) @map("created_at")
  metadata       Json?
  
  discount       Discount @relation(fields: [discountId], references: [id])
  
  @@index([orgId, discountId])
  @@index([orgId, customerId])
  @@index([orgId, resourceType, resourceId])
  @@map("discount_redemptions")
}
```

## Reporting Database Schema

```prisma
// Add to schemas/reporting/schema.prisma

model DiscountRedemption {
  orgId String @map("org_id")
  id    String @default(cuid())

  // Discount data (denormalized)
  discountId String  @map("discount_id")
  name       String
  type       String // "fixed" or "percentage"
  value      Int
  code       String?
  currency   String?
  recurring  String // "once", "forever", "cycles"
  cycles     Int?

  // Customer data (denormalized)
  customerId    String  @map("customer_id")
  customerEmail String? @map("customer_email")
  customerName  String? @map("customer_name")
  
  // Redemption data
  resourceType       String   @map("resource_type") // "subscription", "invoice", "payment", "checkout_session"
  resourceId         String   @map("resource_id")
  amount             Int // Amount saved in smallest currency unit
  redemptionCurrency String   @map("redemption_currency")
  appliedAt          DateTime @map("applied_at")

  // Timestamps
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@id([orgId, id])
  @@index([orgId, discountId])
  @@index([orgId, customerId])
  @@index([orgId, appliedAt])
  @@index([orgId, code])
  @@index([orgId, resourceType, resourceId])
  @@map("discount_redemptions")
}

```

## Integration Points

### Services That Will Use Discounts
1. **Subscription Service**: Apply discounts during subscription creation/renewal
2. **Invoice Service**: Apply discounts before invoice finalization  
3. **Payment Service**: Calculate discounted amounts
4. **Checkout Service**: Validate and apply discount codes during checkout

### Event Topics
Discount events will be published to the following topics:
- `discount.created`
- `discount.updated`
- `discount.deleted`
- `discount.applied`
- `discount.expired`

## Implementation Guidelines

### Validation Rules
1. **Discount codes** must be unique per organization (case-insensitive)
2. **Percentage discounts** must be between 0 and 100
3. **Fixed discounts** require a currency and are stored in smallest currency unit
4. **Cycle-based discounts** require cycles count when recurring is "cycles"
5. **Date ranges** must be valid (starts_at before ends_at)

### Business Logic
1. **Code normalization**: Convert to uppercase for consistency
2. **Redemption tracking**: Count redemptions dynamically from DiscountRedemption table
3. **Expiration checks**: Consider both date range and redemption limits
4. **Currency matching**: Fixed discounts must match transaction currency
5. **Redemption validation**: Query DiscountRedemption count before allowing new redemptions

### Security Considerations
1. **Rate limiting** on discount code validation endpoints
2. **Audit logging** for all discount applications
3. **Org isolation** enforced at all layers

### Performance Considerations
1. **Dynamic redemption counting** ensures data consistency over denormalized counts
2. **Indexed queries** on DiscountRedemption table for efficient counting
3. **Consider caching** discount validation results for high-traffic scenarios
4. **Database query optimization** when checking redemption limits

### Data Sync Strategy
1. **CDC Integration**: Use Change Data Capture to sync discount redemptions to reporting database
2. **Denormalized Data**: Reporting table includes discount and customer data at time of redemption
3. **Historical Preservation**: Reporting database preserves data even after operational deletes
4. **Query-based Analytics**: All metrics aggregated on-demand from detailed redemption data