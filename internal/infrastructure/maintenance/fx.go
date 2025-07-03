package maintenance

import (
	"database/sql"
	"go.uber.org/fx"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
)

var Module = fx.Module("maintenance",
	fx.Provide(
		NewPartitionSchedulerFx,
	),
)

// UsageDBParam is used to inject the usage database connection
type UsageDBParam struct {
	fx.In
	UsageDB *sql.DB `name:"usageDB"`
}

// NewPartitionSchedulerFx creates a new partition scheduler with FX dependency injection
func NewPartitionSchedulerFx(params UsageDBParam, logger logger.Logger, scheduler interfaces.Scheduler) *PartitionScheduler {
	return NewPartitionScheduler(params.UsageDB, logger, scheduler)
}
