package services

import (
	"context"
	"payloop/internal/models"
	"payloop/internal/repository"
	"payloop/internal/repository/tenants"
)

type TenantService struct {
	repository repository.TenantRepository
}

func NewTenantService(repo repository.TenantRepository) TenantService {
	return TenantService{repository: repo}
}

func (s *TenantService) Create(ctx context.Context, input tenants.CreateTenantInput) (models.Tenant, error) {
	return s.repository.Create(ctx, input)
}
