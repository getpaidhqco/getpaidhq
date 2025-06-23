# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

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