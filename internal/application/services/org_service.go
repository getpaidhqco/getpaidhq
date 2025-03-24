package services

import (
	"context"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orgs"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type OrgService struct {
	orgRepository     repositories.OrgRepository
	apiKeyRepository  repositories.ApiKeyRepository
	settingRepository repositories.SettingRepository
	logger            logger.Logger
}

func NewOrgService(
	repo repositories.OrgRepository,
	settingRepository repositories.SettingRepository,
	apiKeyRepository repositories.ApiKeyRepository,
	logger logger.Logger,
) OrgService {
	return OrgService{
		orgRepository:     repo,
		settingRepository: settingRepository,
		apiKeyRepository:  apiKeyRepository,
		logger:            logger,
	}
}

func (s OrgService) Create(ctx context.Context, input orgs.CreateOrgInput) (entities.Org, error) {
	s.logger.Debug("Creating tenant", "input", input)

	id := lib.GenerateId("org")
	org, err := s.orgRepository.Create(ctx, entities.Org{
		Id:          id,
		Name:        input.Name,
		Status:      entities.OrgStatusTrial,
		Country:     input.Country,
		Description: input.Description,
		Metadata:    input.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to create org", "err", err)
		return entities.Org{}, err
	}

	s.logger.Debug("Org created", "org_id", id)
	s.logger.Debug("Creating API key")
	key := lib.GenerateId("sk")
	_, err = s.apiKeyRepository.Create(ctx, entities.ApiKey{
		OrgId:     id,
		Id:        key,
		Key:       key,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to create API key", "org_id", id, "err", err)
		return entities.Org{}, err
	}

	return org, err
}
