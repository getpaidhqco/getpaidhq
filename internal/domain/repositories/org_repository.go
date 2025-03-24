package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type OrgRepository interface {
	Create(ctx context.Context, entity entities.Org) (entities.Org, error)
}
