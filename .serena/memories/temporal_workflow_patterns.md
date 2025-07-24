# Temporal Workflow Patterns in Payloop

## Activity Design Rules
- **Thin Coordinators**: Activities should delegate to domain services
- **No Orchestration**: Never include orchestration services in activities
- **Domain Service Delegation**: Activities call domain services (SubscriptionService, PaymentService)
- **Business Logic Placement**: Keep business logic in domain services, not activities

## Workflow Structure
```go
// Good: Activity delegates to domain service
func (a *BillingActivity) ProcessSubscriptionCharge(ctx context.Context, subscriptionId string) error {
    return a.subscriptionService.ProcessCharge(ctx, subscriptionId)
}

// Bad: Activity contains business logic
func (a *BillingActivity) ProcessSubscriptionCharge(ctx context.Context, subscriptionId string) error {
    // Complex billing logic here - belongs in domain service
}
```

## Key Workflows
- **Subscription Charging**: Automated billing cycles
- **Dunning Management**: Payment failure recovery
- **Webhook Delivery**: Reliable outgoing webhook system
- **Order Processing**: Complex order fulfillment flows

## Temporal Configuration
- **Namespace**: `subscriptions` (created with `temporal operator namespace create -n subscriptions`)
- **Location**: Workflows in `internal/infrastructure/workflow/temporal/workflows/`
- **Activities**: Defined in `internal/infrastructure/workflow/temporal/activities/`

## Testing Patterns
- Use temporal test suite for workflow testing
- Mock domain services in activity tests
- Integration tests available in `internal/infrastructure/workflow/temporal/`