package services

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orgs"
	"payloop/internal/domain/repositories"
)

type OrgService struct {
	repository repositories.OrgRepository
}

func NewOrgService(repo repositories.OrgRepository) OrgService {
	return OrgService{repository: repo}
}

func (s *OrgService) Create(ctx context.Context, input orgs.CreateOrgInput) (entities.Org, error) {
	return s.repository.Create(ctx, input)
}
