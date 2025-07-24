# Payloop Codebase Structure

## Root Directory Structure
```
payloop/
├── main.go                    # Application entry point
├── go.mod/go.sum             # Go module dependencies
├── package.json              # Node.js dependencies (for Prisma)
├── CLAUDE.md                 # Development guidelines and instructions
├── prisma/                   # Main database schema and migrations
├── schemas/                  # Additional schemas (reporting, usage)
├── internal/                 # Core application code
├── assets/                   # Static assets (templates, etc.)
├── docs/                     # Documentation
├── specs/                    # Specifications
├── scripts/                  # Build and deployment scripts
├── docker/                   # Docker configuration
├── .github/                  # GitHub workflows
└── buildspec.yml            # AWS CodeBuild configuration
```

## Internal Architecture (DDD Layers)

### 1. API Layer (`internal/api/`)
- **Controllers**: Handle HTTP requests and responses
- **Routes**: Define API endpoints  
- **Middlewares**: Process requests (authentication, logging, etc.)
- **DTOs**: Data transfer objects for API requests and responses
- **Mappers**: Convert between API DTOs and Application DTOs

### 2. Application Layer (`internal/application/`)
- **Services**: Implement business logic and orchestrate domain operations
- **Interfaces**: Define contracts for repositories and external services
- **DTOs**: Internal data transfer objects
- **Bootstrap**: Application startup and dependency injection configuration

### 3. Domain Layer (`internal/domain/`)
- **Entities**: Core business objects (Subscription, Customer, Payment, etc.)
- **Repositories**: Interfaces for data access
- **Value Objects**: Immutable objects representing concepts
- **Factories**: Create complex domain objects
- **Common**: Shared types and enums (Currency, Country, etc.)

### 4. Infrastructure Layer (`internal/infrastructure/`)
- **Database** (`db/`): PostgreSQL with dual database architecture
- **Authentication** (`authn/`): Clerk, Cognito, and API key implementations
- **Authorization** (`authz/`): Cedar policy engine
- **Payment Providers** (`payments/`): Paystack, Checkout.com
- **Storage** (`storage/`): AWS S3 for document storage
- **Security** (`vault/`): Token vault with encryption
- **Email** (`email/`): Loops integration
- **Cache** (`cache/`): Redis implementation
- **Pub/Sub** (`events/`): NATS for event messaging
- **Queue** (`queue/`): AWS SQS for background jobs
- **Workflow** (`workflow/`): Temporal for business process orchestration
- **Scheduler** (`scheduler/`): Cron for recurring tasks

## Key Patterns
- **Module System**: Each infrastructure component has its own module
- **Factory Pattern**: Used for creating domain objects and external service clients
- **Repository Pattern**: Interfaces in domain, implementations in infrastructure
- **Clean Architecture**: Strict layer separation with dependency inversion