# Payloop

Payloop is a smart recurring payment processing framework designed to provide flexible and extensible subscription management capabilities. It allows developers to easily integrate subscription billing into their applications with support for various payment providers, authentication methods, and workflow orchestration.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [API Endpoints](#api-endpoints)
- [Data Model](#data-model)
- [Integrations](#integrations)
- [Authentication & Authorization](#authentication--authorization)
- [Installation](#installation)
- [Configuration](#configuration)
  - [AWS S3 Configuration](#aws-s3-configuration)
  - [Security Vault Configuration](#security-vault-configuration)
  - [Email Service Configuration](#email-service-configuration)
  - [PDF Generation System](#pdf-generation-system)
  - [Model Context Protocol (MCP) Integration](#model-context-protocol-mcp-integration)
- [Database Migrations](#database-migrations)
- [Development](#development)
  - [Development Commands](#development-commands)
  - [Change Data Capture (CDC)](#change-data-capture-cdc)
- [Deployment](#deployment)

## Overview

Payloop is a comprehensive payment processing system that focuses on subscription management. It provides:

- **Subscription Management**: Complete lifecycle management with billing anchors, pause/resume, and recovery workflows
- **Payment Processing**: Multi-provider support (Paystack, Checkout.com) with secure vault encryption
- **Invoice System**: Automated PDF generation with ChromeDP and Liquid templates
- **Customer Management**: Cohort-based segmentation with secure payment method storage
- **Document Storage**: AWS S3 integration for secure PDF storage and retrieval
- **Email Integration**: Transactional email notifications via Loops
- **Security Vault**: Encrypted storage for sensitive payment tokens using AES or AWS Secrets Manager
- **Dual Database Architecture**: Operational and reporting databases with CDC synchronization
- **AI Integration**: Model Context Protocol (MCP) for AI-friendly invoice operations
- **Webhook System**: Reliable outgoing webhook delivery via Temporal workflows
- **Comprehensive Reporting**: Dedicated analytics database with real-time synchronization

The system is built using Domain-Driven Design (DDD) principles and follows a clean architecture approach with extensive dependency injection.

## Architecture

Payloop follows a layered architecture based on DDD principles:

### Layers

1. **API Layer** (`internal/api/`)
   - Controllers: Handle HTTP requests and responses
   - Routes: Define API endpoints
   - Middlewares: Process requests (authentication, logging, etc.)
   - DTOs: Data transfer objects for API requests and responses

2. **Application Layer** (`internal/application/`)
   - Services: Implement business logic and orchestrate domain operations
   - Interfaces: Define contracts for repositories and external services
   - DTOs: Internal data transfer objects

3. **Domain Layer** (`internal/domain/`)
   - Entities: Core business objects (Subscription, Customer, Payment, etc.)
   - Repositories: Interfaces for data access
   - Value Objects: Immutable objects representing concepts
   - Factories: Create complex domain objects

4. **Infrastructure Layer** (`internal/infrastructure/`)
   - **Database**: PostgreSQL with dual database architecture (operational + reporting)
   - **Authentication**: Clerk (active), Cognito, and API key implementations
   - **Authorization**: Cedar policy engine with role-based access control
   - **Payment Providers**: Paystack (primary), Checkout.com (secondary)
   - **Storage**: AWS S3 for document and PDF storage
   - **Security**: Token vault with AES/AWS Secrets Manager encryption
   - **Email**: Loops integration for transactional emails
   - **Cache**: Redis implementation for performance optimization
   - **Pub/Sub**: NATS for event messaging and workflow coordination
   - **Queue**: AWS SQS for background job processing
   - **Workflow**: Temporal for complex business process orchestration
   - **Scheduler**: Cron implementation for recurring tasks

### Key Components

- **Runtime**: Go 1.24 with advanced dependency injection via Uber FX
- **Web Framework**: Gin HTTP router with comprehensive middleware support
- **Database System**: 
  - **Primary**: PostgreSQL operational database with Prisma ORM
  - **Reporting**: Dedicated analytics database with CDC synchronization
  - **Schema Management**: Prisma with automated migrations
- **PDF Generation**: ChromeDP (headless Chrome) with Liquid templating
- **Document Storage**: AWS S3 with server-side encryption and presigned URLs
- **Security Vault**: Multi-provider encryption (AES, AWS Secrets Manager)
- **Workflow Engine**: Temporal for complex business process orchestration
- **Event System**: NATS for pub/sub messaging and real-time coordination
- **Caching**: Redis for performance optimization and session management
- **Authorization**: Cedar policy engine with granular role-based permissions
- **Email Service**: Loops integration for transactional notifications
- **AI Integration**: Model Context Protocol (MCP) server for AI-friendly operations

## API Endpoints

Payloop exposes a RESTful API with the following main endpoints:

### Subscriptions

- `GET /api/subscriptions`: List subscriptions
- `GET /api/subscriptions/:id`: Get subscription details
- `GET /api/subscriptions/:id/payments`: List payments for a subscription
- `PUT /api/subscriptions/:id/pause`: Pause a subscription
- `PUT /api/subscriptions/:id/cancel`: Cancel a subscription
- `PUT /api/subscriptions/:id/resume`: Resume a paused subscription
- `PUT /api/subscriptions/:id/billing-anchor`: Update billing anchor date
- `PATCH /api/subscriptions/:id`: Update subscription details

### Customers

- `POST /api/customers`: Create a new customer
- `POST /api/customers/:id/payment-methods`: Add a payment method to a customer
- `PUT /api/customers/:id/payment-methods/:pmid`: Update a customer's payment method

### Payment Methods

- `GET /api/payment-methods/:id`: Get payment method details

### Other Endpoints

The API also includes endpoints for:
- Orders
- Products
- Organizations
- Users
- Carts
- Sessions
- Webhooks
- Reports
- Payment Service Providers (PSPs)

## Data Model

Payloop's data model includes the following key entities:

### Core Entities

- **Org**: Organizations that use the system
- **User**: Users who can access the system
- **ApiKey**: API keys for authentication
- **Document**: File storage references with metadata

### Product Catalog

- **Product**: Products that can be sold
- **Variant**: Product variants with configuration options
- **Price**: Pricing information for variants with tax handling

### Sales

- **Cart**: Shopping carts with item management
- **Session**: User sessions with state tracking
- **Order**: Customer orders with fulfillment status
- **OrderItem**: Items in an order with quantity and pricing

### Customers

- **Customer**: Customer information with profile data
- **PaymentMethod**: Customer payment methods (insecure references)
- **SecurePaymentMethod**: Vault-encrypted payment method storage
- **Cohort**: Customer groupings for analytics and targeting

### Billing

- **Subscription**: Recurring billing subscriptions with lifecycle management
- **Invoice**: Billing documents with PDF generation and storage
- **InvoiceHistory**: Audit trail of invoice changes
- **Payment**: Payment transactions with provider integration
- **Refund**: Payment refunds with accounting reconciliation

### Integration

- **WebhookSubscription**: Webhook configuration for event notifications

## Integrations

Payloop integrates with various external systems:

### Authentication Providers
- Cognito
- Clerk
- API Key

### Payment Providers
- Paystack

### Infrastructure Services
- Temporal (Workflow Engine)
- NATS (Pub/Sub)
- AWS SQS (Queue)
- Redis (Cache)
- PostgreSQL (Database)
- Cedar (Authorization)

## Authentication & Authorization

Payloop uses a flexible authentication system that supports multiple providers:

- **API Key**: For server-to-server authentication
- **Cognito**: AWS Cognito for user authentication
- **Clerk**: Clerk.dev for user authentication

Authorization is handled by Cedar, a policy-based access control system. Policies are defined in the `policy.cedar` file.

To enable or disable an authentication method, add or remove the `group:"authenticators"` FX tag from the
injection, or remove the FX DI in modules.go.

## Installation

### Prerequisites

- **Docker & Docker Compose**: For running required services locally
- **Go 1.24+**: Latest Go runtime with modules support
- **Node.js & pnpm**: For Prisma database management
- **Temporal CLI**: For workflow namespace management
- **AWS CLI**: For S3 storage configuration (production)
- **Chrome/Chromium**: For PDF generation (automatically handled in Docker)

### Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd payloop
   ```

2. **Install dependencies**
   ```bash
   go mod download
   pnpm install  # For Prisma
   ```

3. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Start infrastructure services**
   ```bash
   docker-compose up -d
   ```

5. **Set up databases**
   ```bash
   # Generate Prisma client
   pnpm dlx prisma generate

   # Push schema to development database
   pnpm dlx prisma db push

   # Push reporting schema
   pnpm dlx prisma db push --schema=schemas/reporting/schema.prisma
   ```

6. **Create Temporal namespace**
   ```bash
   temporal operator namespace create -n subscriptions
   ```

7. **Seed initial data**
   ```bash
   node prisma/seed.js
   ```

8. **Start the application**
   ```bash
   go run main.go serve
   ```

## Configuration

Configuration is managed through the `.env` file with the `GETPAIDHQ_` prefix convention. Key configuration areas include:

- **Server**: Port, host, and MCP SSE port configuration
- **Database**: Dual database URLs (operational and reporting)
- **Logging**: Level, format, and output configuration
- **Authentication**: Clerk, Cognito, and API key settings
- **Payment Providers**: Paystack and Checkout.com credentials
- **Security Vault**: AES encryption keys and AWS Secrets Manager settings
- **Email Service**: Loops API configuration
- **File Storage**: AWS S3 bucket and region settings
- **Infrastructure**: NATS, Redis, SQS, and Temporal configuration
- **CDC**: Change Data Capture settings for database synchronization

### AWS S3 Configuration

The document storage service requires S3 access for storing invoice PDFs and other documents. When running on ECS, ensure your ECS task role has the following IAM permissions:

#### Required S3 Permissions

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject",
                "s3:PutObjectAcl"
            ],
            "Resource": "arn:aws:s3:::your-document-bucket/*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:ListBucket",
                "s3:GetBucketLocation"
            ],
            "Resource": "arn:aws:s3:::your-document-bucket"
        }
    ]
}
```

#### Environment Variables

Set the following environment variables for S3 configuration:

```bash
# S3 Configuration
S3_BUCKET=your-document-bucket
S3_REGION=us-east-1
```

#### ECS Task Role Setup

1. Create an IAM role for your ECS task
2. Attach the above policy to the role
3. Assign the role to your ECS task definition
4. The AWS SDK will automatically use the role credentials

#### Security Considerations

- Documents are encrypted at rest using AES-256 server-side encryption
- Private documents use access tokens and presigned URLs for secure access
- Bucket should have proper CORS configuration if accessed from web clients
- Consider enabling S3 bucket versioning for document history

### Security Vault Configuration

Payloop includes a secure vault system for encrypting sensitive payment tokens and data.

#### AES Vault (Default)
```bash
# Generate a 32-byte AES key
GETPAIDHQ_TOKEN_VAULT_TYPE=aes
GETPAIDHQ_TOKEN_VAULT_AES_KEY=$(openssl rand -base64 32)
```

#### AWS Secrets Manager Vault
```bash
GETPAIDHQ_TOKEN_VAULT_TYPE=aws_secrets_manager
GETPAIDHQ_TOKEN_VAULT_AWS_REGION=us-east-1
GETPAIDHQ_TOKEN_VAULT_AWS_PATH=payloop/payment-tokens
```

### Email Service Configuration

Configure Loops for transactional email delivery:

```bash
# Email Service Configuration
GETPAIDHQ_EMAIL_PROVIDER=loops
LOOPS_API_KEY=your_loops_api_key
GETPAIDHQ_LOOPS_API_ENDPOINT=https://api.loops.so/v1/transactional
GETPAIDHQ_EMAIL_FROM_EMAIL=invoices@yourdomain.com
GETPAIDHQ_EMAIL_FROM_NAME=Your Company Name
```

### Authentication Configuration

Payloop supports multiple authentication providers including Clerk, Cognito, and API keys.

#### Clerk Authentication & OAuth 2.1
```bash
# Clerk Authentication Configuration
GPHQ_CLERK_SECRET=your_clerk_secret_key
GPHQ_CLERK_DOMAIN=your-app.clerk.accounts.dev

# Example domains:
# For development: your-app.clerk.accounts.dev
# For production: clerk.yourdomain.com
```

The `GPHQ_CLERK_DOMAIN` is required for:
- OAuth 2.1 discovery endpoints (`/.well-known/oauth-authorization-server`)
- JWKS token validation for MCP authentication
- Proper redirection to Clerk's authorization server

#### API Key Authentication
```bash
# API keys are generated automatically and stored in the database
# No additional configuration required
```

#### AWS Cognito Authentication
```bash
GPHQ_COGNITO_CLIENT_ID=your_cognito_client_id
GPHQ_COGNITO_POOL_ID=your_cognito_pool_id
GPHQ_COGNITO_REGION=us-east-1
```

### PDF Generation System

The system uses ChromeDP (headless Chrome) for high-quality PDF generation:

- **Templates**: Liquid templates located in `assets/templates/invoices/`
- **Fonts**: Custom fonts can be added to support various languages
- **Styling**: CSS-based styling with full Chrome rendering capabilities
- **Performance**: Automatic resource management and memory cleanup
- **Storage**: Generated PDFs are automatically uploaded to S3

#### Template Customization
1. Modify templates in `assets/templates/invoices/`
2. Use Liquid syntax for dynamic content
3. Test with `go test -v ./internal/application/lib/pdf/...`

### Model Context Protocol (MCP) Integration

Payloop includes an MCP server for AI-friendly operations:

- **Server**: Runs on port 8084 with Server-Sent Events
- **Tools**: Invoice creation and management operations
- **Location**: `/internal/mcp/`
- **Usage**: Enables AI assistants to interact with Payloop programmatically

#### Available MCP Tools
- `hello_world`: Basic connectivity test
- `create_invoice`: Create invoices with AI assistance
- More tools can be added by implementing the MCP tool interface

## Database Migrations

Payloop uses Prisma to manage database schema and migrations:

- Development: Run migrations locally using Prisma CLI
- Test/Production: Migrations are executed by the CI/CD pipeline before deployment

For the Postgres database we use Prisma to manage the database schema and migrations. Migrations in Test and Prod
environments are managed by the CI/CD pipeline. Migrations are executed before the Payloop backend is built and deployed.
Check the buildspec.yml file for more details.

## Development

### Development Commands

The following commands are available for development and testing:

#### Building and Running
```bash
go run main.go serve                    # Start the API server
docker-compose up -d                    # Start required services
```

#### Database Operations
```bash
pnpm dlx prisma generate                # Generate Prisma client
pnpm dlx prisma db push                 # Push schema changes to development database
pnpm dlx prisma migrate deploy          # Deploy migrations (used in CI/CD)
pnpm dlx prisma format                  # Format Prisma schema files
```

#### Reporting Database
```bash
pnpm dlx prisma format --schema=schemas/reporting/schema.prisma         # Format reporting schema
pnpm dlx prisma db push --schema=schemas/reporting/schema.prisma        # Push reporting schema changes
```

#### Testing
```bash
go test ./...                           # Run all tests
go test ./internal/application/services/...    # Run service layer tests
go test -v ./internal/application/lib/pdf/...  # Run PDF generation tests with verbose output
```

#### Deployment
```bash
pnpm run deploy:test                    # Deploy to test environment
pnpm run deploy:prod                    # Deploy to production environment
```

#### Development Tunnels
```bash
pnpm run tunnel:test                    # Create SSH tunnel to test environment resources
pnpm run tunnel:prod                    # Create SSH tunnel to production environment resources
```

### Change Data Capture (CDC)

Payloop uses two databases:
- `payloop`: Operational database
- `payloop_reporting`: Reporting database

These databases are kept in sync using a Change Data Capture (CDC) process.

Currently (Apr 2025) the CDC library doesn't update the publication records for the logical replication, which means
we need to manually update the publication records in the `pg_publication` table every time there's an update to the
CDC Stream service. We need to remove the current publication and subscription so that the system can create
a new one when the server starts.

```sql
SELECT *
FROM pg_publication;
SELECT *
FROM pg_replication_slots;

DROP PUBLICATION cdc_pub;
SELECT pg_terminate_backend(22081);
SELECT pg_drop_replication_slot('cdc_slot2');
```

### Connecting to Test Environment

Via the bastion:

Payloop API (port 8888->8081):
```
ssh -o StrictHostKeyChecking=no -N -L 8888:payloop.temporal.temporal:8081 ec2-user@ec2-34-244-193-216.eu-west-1.compute.amazonaws.com -i cj-bastion-test.pem -v
```

Temporal UI (port 9999->8080):
```
ssh -o StrictHostKeyChecking=no -N -L 9999:temporal-svc.temporal:8080 ec2-user@ec2-34-244-193-216.eu-west-1.compute.amazonaws.com -i cj-bastion-test.pem -v
```

## Deployment

Payloop is containerized using Docker and can be deployed to various environments:

- The `Dockerfile` defines the build process
- The `buildspec.yml` file defines the CI/CD pipeline for AWS CodeBuild

### Creating ECR Registries

For AWS deployment, ECR registries are used to store Docker images:

```
# Golang base image
docker pull golang:1.24-alpine

aws ecr create-repository --repository-name golang-1_24-alpine --profile=cj-test
aws ecr get-login-password --region eu-west-1 --profile=cj-test |  docker login --username AWS --password-stdin 329237115630.dkr.ecr.eu-west-1.amazonaws.com
docker tag golang:1.24-alpine 329237115630.dkr.ecr.eu-west-1.amazonaws.com/golang-1_24-alpine
docker push 329237115630.dkr.ecr.eu-west-1.amazonaws.com/golang-1_24-alpine
```

```
# Temporal images
docker pull temporalio/auto-setup

aws ecr create-repository --repository-name temporalio_auto_setup --profile=cj-test
aws ecr get-login-password --region eu-west-1 --profile=cj-test |  docker login --username AWS --password-stdin 329237115630.dkr.ecr.eu-west-1.amazonaws.com
docker tag temporalio/auto-setup 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_auto_setup
docker push 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_auto_setup

docker pull temporalio/admin-tools

aws ecr create-repository --repository-name temporalio_admin_tools --profile=cj-test
aws ecr get-login-password --region eu-west-1 --profile=cj-test |  docker login --username AWS --password-stdin 329237115630.dkr.ecr.eu-west-1.amazonaws.com
docker tag temporalio/admin-tools 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_admin_tools
docker push 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_admin_tools

docker pull temporalio/ui
aws ecr create-repository --repository-name temporalio_ui --profile=cj-test
aws ecr get-login-password --region eu-west-1 --profile=cj-test |  docker login --username AWS --password-stdin 329237115630.dkr.ecr.eu-west-1.amazonaws.com
docker tag temporalio/ui 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_ui
docker push 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_ui
```
