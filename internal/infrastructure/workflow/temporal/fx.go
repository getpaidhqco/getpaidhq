package temporal

import (
	"go.uber.org/fx"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/infrastructure/workflow/temporal/workflows"
)

// Module exports dependency
var Module = fx.Options(
	fx.Provide(activities.Module),
	fx.Provide(workflows.NewPaymentSuccessWorkflow),
	fx.Provide(NewTemporalEngine),
)
