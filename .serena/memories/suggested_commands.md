# Payloop Development Commands

## Building and Running
```bash
go run main.go serve                    # Start the API server
docker-compose up -d                    # Start required services (database, temporal, etc.)
```

## Database Operations
```bash
pnpm dlx prisma generate                # Generate Prisma client
pnpm dlx prisma db push                 # Push schema changes to development database
pnpm dlx prisma migrate deploy          # Deploy migrations (used in CI/CD)
pnpm dlx prisma format                  # Format Prisma schema files
```

## Reporting Database
```bash
pnpm dlx prisma format --schema=schemas/reporting/schema.prisma         # Format reporting schema
pnpm dlx prisma db push --schema=schemas/reporting/schema.prisma        # Push reporting schema changes
```

## Testing
```bash
go test ./...                           # Run all tests
go test ./internal/application/services/...    # Run service layer tests
go test -v ./internal/application/lib/pdf/...  # Run PDF generation tests with verbose output
```

## Deployment
```bash
pnpm run deploy:test                    # Deploy to test environment
pnpm run deploy:prod                    # Deploy to production environment
```

## Development Tunnels
```bash
pnpm run tunnel:test                    # Create SSH tunnel to test environment resources
pnpm run tunnel:prod                    # Create SSH tunnel to production environment resources
```

## Temporal Setup
```bash
temporal operator namespace create -n subscriptions    # Create Temporal namespace (one-time setup)
```

## Database Seeding
```bash
node prisma/seed.js                     # Seed initial data
```

## System Utilities (Darwin/macOS)
- `git` - Git version control
- `ls` - List directory contents  
- `cd` - Change directory
- `grep` - Search text patterns (prefer `rg` ripgrep if available)
- `find` - Find files and directories