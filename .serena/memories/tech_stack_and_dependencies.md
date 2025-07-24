# Payloop Tech Stack

## Core Technologies
- **Language**: Go 1.24
- **Web Framework**: Gin HTTP router with comprehensive middleware support
- **Dependency Injection**: Uber FX
- **Database**: PostgreSQL with dual database architecture (operational + reporting)
- **ORM**: Prisma for schema management and migrations
- **Package Manager**: pnpm for Node.js dependencies (used for Prisma)

## Infrastructure Components
- **Cache**: Redis for performance optimization and session management
- **Queue**: AWS SQS for background job processing
- **Pub/Sub**: NATS for event messaging and real-time coordination
- **Workflow Engine**: Temporal for complex business process orchestration
- **Scheduler**: Cron implementation for recurring tasks
- **Storage**: AWS S3 for document and PDF storage with server-side encryption

## Security & Authentication
- **Authentication**: Clerk (active), Cognito, API keys (configurable)
- **Authorization**: Cedar policy engine with granular role-based permissions
- **Security Vault**: Multi-provider encryption (AES, AWS Secrets Manager)

## Specialized Libraries
- **PDF Generation**: ChromeDP (headless Chrome) with Liquid templating system
- **Email Service**: Loops integration for transactional notifications
- **Payment Providers**: Paystack (primary), Checkout.com (secondary)
- **Testing**: testcontainers-go for integration testing
- **Logging**: Zap logger with structured logging
- **Configuration**: Viper for configuration management

## Key Dependencies (from go.mod)
- **Framework**: gin-gonic/gin, spf13/cobra, spf13/viper
- **Database**: jackc/pgx/v5, lib/pq
- **Temporal**: go.temporal.io/sdk, go.temporal.io/api  
- **AWS**: aws-sdk-go-v2 (S3, SQS, Secrets Manager)
- **Authentication**: clerkinc/clerk-sdk-go, dgrijalva/jwt-go
- **Payment**: checkout/checkout-sdk-go, mdwt/paystack-go
- **Other**: redis/go-redis/v9, nats-io/nats.go, cedar-policy/cedar-go