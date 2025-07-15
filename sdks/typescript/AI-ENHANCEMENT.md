# AI SDK Enhancement System

## Overview

The GetPaidHQ TypeScript SDK uses an AI-enhanced development process that goes beyond basic OpenAPI code generation. This system creates developer-friendly abstractions, utilities, and patterns that make the SDK more pleasant to work with.

## How It Works

### 1. Basic Generation (`npm run generate`)
- Uses OpenAPI Generator to create basic TypeScript client from `swagger.yml`
- Generates raw API clients, models, and types
- Output: `src/generated/` directory with basic functionality

### 2. AI Enhancement (`npm run enhance`)
- Reads `scripts/ai-enhancement-context.md` for business domain knowledge
- Creates enhanced wrappers and utilities based on best practices
- Output: `src/enhanced/` directory with developer-friendly features

### 3. Full Generation (`npm run generate:full`)
- Runs both generate and enhance steps
- Creates a complete SDK with both raw API access and enhanced features

## Generated Enhancements

### Enhanced Client (`src/enhanced/client.ts`)
```typescript
import { GetPaidHQ } from '@getpaidhq/sdk'

const client = new GetPaidHQ({
  apiKey: 'your-api-key',
  enableRetries: true,
  maxRetries: 3,
  debug: true
})

// Simple subscription creation
const subscription = await client.quickSubscription(
  'customer-id',
  'price-id',
  { trialDays: 14 }
)
```

### Builder Patterns (`src/enhanced/builders/`)
```typescript
// Complex subscription with fluent API
const subscription = await client.createSubscription()
  .customer('cus_123')
  .addItem('price_base', 1)
  .addItem('price_addon', 3)
  .withTrial(30)
  .withMetadata({ source: 'dashboard' })
  .create()
```

### Utility Classes (`src/enhanced/utilities/`)
- **AmountFormatter**: Handle currency conversions (dollars ↔ cents)
- **WebhookVerifier**: Secure webhook signature verification
- **UsageRecorder**: Batch usage recording with auto-flush
- **RetryWrapper**: Automatic retry logic with exponential backoff

### Async Iterators (`src/enhanced/iterators/`)
```typescript
// Iterate through all subscriptions without pagination logic
for await (const subscription of client.subscriptions.listAll()) {
  console.log(subscription.id)
}

// Filter and transform with async methods
const activeSubscriptions = await client.subscriptions
  .listAll()
  .filter(sub => sub.status === 'active')
  .toArray()
```

### Usage Examples (`src/enhanced/examples/`)
- Quick start guide with common use cases
- Complex scenarios (usage-based billing, webhooks)
- Error handling patterns
- Best practices demonstrations

## Development Workflow

### Standard Workflow
```bash
# After updating swagger.yml or AI context
npm run generate:full  # Generate + enhance
npm run build          # Build the final SDK
npm test              # Run tests
```

### Development Iteration
```bash
# When working on enhancements
npm run enhance       # Only run enhancement step
npm run typecheck     # Verify TypeScript
```

### AI Enhancement Context

The `scripts/ai-enhancement-context.md` file provides:

1. **Business Domain Knowledge**: Subscription billing concepts, pricing models
2. **Enhancement Guidelines**: Developer experience patterns to implement
3. **Common Use Cases**: Real-world scenarios the SDK should optimize for
4. **API Patterns**: RESTful conventions and builder patterns
5. **Pain Points**: Known developer challenges to solve

## Customizing Enhancements

### Adding New Utilities
1. Add utility class to `scripts/enhance.ts` in `createUtilities()`
2. Export from enhanced index file
3. Add usage examples

### Modifying Builder Patterns
1. Update builder creation in `createBuilders()`
2. Follow fluent API conventions
3. Add TypeScript type safety

### Adding Business Logic
1. Update `ai-enhancement-context.md` with new domain concepts
2. Implement patterns in `scripts/enhance.ts`
3. Create examples demonstrating usage

## Architecture Benefits

### Two-Layer Approach
- **Raw Layer** (`src/generated/`): Direct API access for advanced users
- **Enhanced Layer** (`src/enhanced/`): Developer-friendly wrappers for common use cases

### Gradual Adoption
- Developers can use enhanced features selectively
- Fall back to raw API when needed
- No magic - enhanced layer is transparent

### AI-Driven Development
- Business domain knowledge guides enhancement decisions
- Consistent patterns across all resources
- Automatic best practice implementation

## Future Enhancements

The AI enhancement system can be extended to add:

- **Validation Helpers**: Client-side validation before API calls
- **Caching Layer**: Intelligent response caching
- **Metrics Collection**: Built-in usage analytics
- **Error Recovery**: Automatic error handling strategies
- **Development Tools**: Testing utilities and mock clients

## Maintenance

### Updating Business Logic
1. Modify `scripts/ai-enhancement-context.md`
2. Run `npm run enhance`
3. Review generated enhancements
4. Test with real use cases

### Adding New API Endpoints
1. Update OpenAPI specification (`swagger.yml`)
2. Run `npm run generate:full`
3. Enhancements automatically pick up new endpoints
4. Add specific patterns if needed

This system ensures the SDK evolves with both API changes and developer needs, providing a consistently excellent developer experience.