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
- [Database Migrations](#database-migrations)
- [Development](#development)
- [Deployment](#deployment)

## Overview

Payloop is a comprehensive payment processing system that focuses on subscription management. It provides:

- Subscription creation and management
- Payment processing with multiple providers
- Customer management
- Product and pricing configuration
- Order processing
- Webhook integrations
- Reporting capabilities

The system is built using Domain-Driven Design (DDD) principles and follows a clean architecture approach.

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
   - Database: PostgreSQL implementation of repositories
   - Authentication: API key, Cognito, and Clerk implementations
   - Authorization: Cedar policy engine
   - Payment Providers: Paystack integration
   - Cache: Redis implementation
   - Pub/Sub: NATS implementation
   - Queue: AWS SQS implementation
   - Workflow: Temporal implementation
   - Scheduler: Cron implementation

### Key Components

- **Dependency Injection**: Uses Uber's FX library for dependency injection
- **Web Framework**: Uses Gin for HTTP routing and handling
- **Database**: PostgreSQL with Prisma for schema management
- **Workflow Engine**: Temporal for orchestrating complex workflows
- **Event System**: NATS for pub/sub messaging
- **Caching**: Redis for caching
- **Authorization**: Cedar for policy-based access control

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

### Product Catalog

- **Product**: Products that can be sold
- **Variant**: Product variants
- **Price**: Pricing information for variants

### Sales

- **Cart**: Shopping carts
- **Session**: User sessions
- **Order**: Customer orders
- **OrderItem**: Items in an order

### Customers

- **Customer**: Customer information
- **PaymentMethod**: Customer payment methods
- **Cohort**: Customer groupings

### Billing

- **Subscription**: Recurring billing subscriptions
- **Payment**: Payment transactions
- **Refund**: Payment refunds

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

- Docker
- Docker Compose
- Go 1.24
- Temporal client

### Setup

1. Clone the repository
2. Run Docker Compose to start the required services:
   ```
   docker-compose up -d
   ```
3. Create the `subscriptions` namespace in Temporal:
   ```
   temporal operator namespace create -n subscriptions
   ```
4. Run the seed script to create initial data

## Configuration

Configuration is managed through the `config.yml` file, which includes settings for:

- Server (port, host)
- Database connection
- Logging
- Authentication
- Payment providers
- Pub/Sub
- Subscriptions

## Database Migrations

Payloop uses Prisma to manage database schema and migrations:

- Development: Run migrations locally using Prisma CLI
- Test/Production: Migrations are executed by the CI/CD pipeline before deployment

For the Postgres database we use Prisma to manage the database schema and migrations. Migrations in Test and Prod
environments are managed by the CI/CD pipeline. Migrations are executed before the Payloop backend is built and deployed.
Check the buildspec.yml file for more details.

## Development

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
