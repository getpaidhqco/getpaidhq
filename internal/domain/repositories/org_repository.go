package repositories

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orgs"
)

type OrgRepository interface {
	Create(ctx context.Context, input orgs.CreateOrgInput) (entities.Org, error)
}
