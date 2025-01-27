package repositories

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/orgs"
)

type OrgRepository interface {
	Create(ctx context.Context, input orgs.CreateOrgInput) (entities.Org, error)
}
