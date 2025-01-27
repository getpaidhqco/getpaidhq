package services

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/orgs"
	"payloop/internal/repository"
)

type OrgService struct {
	repository repository.OrgRepository
}

func NewOrgService(repo repository.OrgRepository) OrgService {
	return OrgService{repository: repo}
}

func (s *OrgService) Create(ctx context.Context, input orgs.CreateOrgInput) (entities.Org, error) {
	return s.repository.Create(ctx, input)
}
