# GoLand Development Guidelines
- use context7
- ALWAYS LOOK AT OTHER FILES IN THE PROJECT FOR EXAMPLES AND IMPLEMENT THE SAME WAY

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


