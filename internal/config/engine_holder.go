package config

import (
	"context"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

// engineHolder breaks the construction-time cycle between services, activities,
// and the workflow engine: services need an engine reference, the engine needs
// activities, and activities need the services. Services hold the holder at
// construction; the real engine is plugged into inner once it has been built.
type engineHolder struct {
	inner port.Engine
}

func (e *engineHolder) StartWorkflow(ctx context.Context, id port.WorkflowType, payload interface{}) (port.WorkflowResult, error) {
	return e.inner.StartWorkflow(ctx, id, payload)
}

func (e *engineHolder) StartSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	return e.inner.StartSubscriptionWorkflow(ctx, sub)
}

func (e *engineHolder) UpdateSubscriptionWorkflow(ctx context.Context, updateName string, sub domain.Subscription) error {
	return e.inner.UpdateSubscriptionWorkflow(ctx, updateName, sub)
}

func (e *engineHolder) CancelSubscriptionWorkflow(ctx context.Context, sub domain.Subscription) error {
	return e.inner.CancelSubscriptionWorkflow(ctx, sub)
}

func (e *engineHolder) SignalSubscriptionWorkflow(ctx context.Context, signal string, sub domain.Subscription, payload interface{}) error {
	return e.inner.SignalSubscriptionWorkflow(ctx, signal, sub, payload)
}
