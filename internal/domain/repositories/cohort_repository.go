package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type CohortRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Cohort, error)
	Create(ctx context.Context, input entities.Cohort) (entities.Cohort, error)
	Update(ctx context.Context, input entities.Cohort) (entities.Cohort, error)
	Delete(ctx context.Context, input entities.Cohort) (entities.Cohort, error)
}
