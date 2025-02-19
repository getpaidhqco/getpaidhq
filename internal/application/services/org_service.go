package services

import (
	"context"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orgs"
	"payloop/internal/domain/repositories"
)

type OrgService struct {
	orgRepository     repositories.OrgRepository
	settingRepository repositories.SettingRepository
	logger            logger.Logger
}

func NewOrgService(
	repo repositories.OrgRepository,
	settingRepository repositories.SettingRepository,
	logger logger.Logger,
) OrgService {
	return OrgService{
		orgRepository:     repo,
		settingRepository: settingRepository,
		logger:            logger,
	}
}

func (s OrgService) Create(ctx context.Context, input orgs.CreateOrgInput) (entities.Org, error) {
	return s.orgRepository.Create(ctx, input)
}
