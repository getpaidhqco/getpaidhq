# Task Completion Guidelines for Payloop

## After Completing Any Task

### 1. Testing
- **Run Tests**: Execute `go test ./...` to run all tests
- **Service Tests**: Run `go test ./internal/application/services/...` for service layer tests
- **Specific Tests**: Use `go test -v ./path/to/specific/...` for focused testing
- **PDF Tests**: Run `go test -v ./internal/application/lib/pdf/...` for PDF generation tests

### 2. Code Quality (No automatic linting/formatting found)
Since there are no Makefile or automatic linting commands found, ensure:
- Code follows Go conventions and existing patterns
- Domain-Driven Design principles are maintained
- DTO layer separation rules are strictly followed
- Multi-tenancy patterns are preserved (orgId inclusion)

### 3. Database Operations
If database changes were made:
```bash
pnpm dlx prisma format                  # Format schema files
pnpm dlx prisma generate                # Regenerate Prisma client
pnpm dlx prisma db push                 # Push changes to development database
```

For reporting database changes:
```bash
pnpm dlx prisma format --schema=schemas/reporting/schema.prisma
pnpm dlx prisma db push --schema=schemas/reporting/schema.prisma
```

### 4. Integration Testing
- Ensure services start correctly: `docker-compose up -d` followed by `go run main.go serve`
- Test any new endpoints or functionality manually
- Verify multi-tenant isolation works correctly

### 5. Documentation Updates
- Update relevant documentation in `docs/` if new features were added
- Update API documentation if endpoints were modified
- Ensure CLAUDE.md guidelines are followed

## Important Verification Steps
1. **Architecture Compliance**: Verify no cross-layer dependencies were introduced
2. **DTO Separation**: Confirm API DTOs are not used in application services
3. **Entity Validation**: Ensure entity construction uses proper factory methods
4. **Multi-tenancy**: Verify `orgId` is properly included in all operations
5. **Error Handling**: Check that errors are properly wrapped and logged

## Common Patterns to Maintain
- Use existing codebase patterns as templates
- Follow entity construction patterns with factory methods
- Maintain repository interface definitions in domain layer
- Keep business logic in domain services, not activities (for Temporal workflows)