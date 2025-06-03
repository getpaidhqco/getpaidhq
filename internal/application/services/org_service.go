package services

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/authn"
	pubsub "payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type OrgService struct {
	pubsub            pubsub.PubSub
	authProvider      authn.AuthProvider
	orgRepository     repositories.OrgRepository
	cohortRepository  repositories.CohortRepository
	apiKeyRepository  repositories.ApiKeyRepository
	settingRepository repositories.SettingRepository
	logger            logger.Logger
}

func NewOrgService(
	repo repositories.OrgRepository,
	pubsub pubsub.PubSub,
	authProvider authn.AuthProvider,
	cohortRepository repositories.CohortRepository,
	settingRepository repositories.SettingRepository,
	apiKeyRepository repositories.ApiKeyRepository,
	logger logger.Logger,
) OrgService {
	return OrgService{
		authProvider:      authProvider,
		orgRepository:     repo,
		pubsub:            pubsub,
		cohortRepository:  cohortRepository,
		settingRepository: settingRepository,
		apiKeyRepository:  apiKeyRepository,
		logger:            logger,
	}
}

func (s OrgService) Create(ctx context.Context, input dto.CreateOrgInput) (entities.Org, error) {
	s.logger.Debug("Creating tenant", "input", input)

	id := lib.GenerateId("org")
	org, err := s.orgRepository.Create(ctx, entities.Org{
		Id:        id,
		Name:      input.Name,
		Status:    entities.OrgStatusActive,
		Country:   input.Country,
		Timezone:  input.Timezone,
		Metadata:  input.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

	cohorts := []string{"signup_date"}
	for _, cohort := range cohorts {
		s.logger.Debugf("Creating cohort [%s]", cohort)
		_, err = s.cohortRepository.Create(ctx, entities.Cohort{
			OrgId:     id,
			Id:        cohort,
			Name:      cohort,
			Type:      entities.CohortType(cohort),
			Metadata:  nil,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		})
		if err != nil {
			s.logger.Warn("Failed to create cohort", "org_id", id, "cohort", cohort, "err", err)
		}
	}

	if input.Owner.Id != "" {
		s.logger.Debug("Creating auth provider org")
		err = s.authProvider.CreateOrg(org, input.Owner.Id)
	}
	
	_ = s.pubsub.Publish(id, topic.OrgCreated, org)

	return org, err
}
