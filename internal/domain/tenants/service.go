package tenants

import (
	"context"
	"payloop/internal/models"
)

type Service struct {
	repository Repository
}

func NewTenantService(repo Repository) Service {
	return Service{repository: repo}
}

func (s *Service) Create(ctx context.Context, input CreateTenantInput) (models.Tenant, error) {
	return s.repository.Create(ctx, input)
}
