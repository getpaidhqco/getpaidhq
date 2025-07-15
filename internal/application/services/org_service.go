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
	pubsub             pubsub.NotificationPublisher
	authProvider       authn.AuthProvider
	orgRepository      repositories.OrgRepository
	cohortRepository   repositories.CohortRepository
	apiKeyRepository   repositories.ApiKeyRepository
	settingRepository  repositories.SettingRepository
	metadataRepository repositories.MetadataStoreRepository
	logger             logger.Logger
}

func NewOrgService(
	repo repositories.OrgRepository,
	pubsub pubsub.NotificationPublisher,
	authProvider authn.AuthProvider,
	cohortRepository repositories.CohortRepository,
	settingRepository repositories.SettingRepository,
	metadataRepository repositories.MetadataStoreRepository,
	apiKeyRepository repositories.ApiKeyRepository,
	logger logger.Logger,
) OrgService {
	return OrgService{
		authProvider:       authProvider,
		orgRepository:      repo,
		pubsub:             pubsub,
		cohortRepository:   cohortRepository,
		settingRepository:  settingRepository,
		apiKeyRepository:   apiKeyRepository,
		metadataRepository: metadataRepository,
		logger:             logger,
	}
}

func (s OrgService) Create(ctx context.Context, input dto.CreateOrgInput) (entities.Org, error) {
	s.logger.Debug("Creating tenant", "input", input)

	org := entities.Org{
		Id:        lib.GenerateId("org"),
		Name:      input.Name,
		Status:    entities.OrgStatusActive,
		Country:   input.Country,
		Timezone:  input.Timezone,
		Metadata:  input.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// TODO think about this for e.g. in the Clerk case:
	// We need to create an org in Clerk, and if we do it separately then we need to do a lookup every time to
	// get the local org id.  If we reuse the Clerk org id as the local org id, then we can avoid this lookup.
	// Clerk org ids format is the same as our org ids, so we can use the same id.
	if input.Owner.Id != "" {
		s.logger.Debug("Creating auth provider org")
		extOrg, err := s.authProvider.CreateOrg(ctx, org, input.Owner.Id)
		if err != nil {
			s.logger.Error("Failed to create org in auth provider", "err", err)
			return entities.Org{}, err
		}
		org.Id = extOrg.ExternalId
	}

	org, err := s.orgRepository.Create(ctx, org)
	if err != nil {
		s.logger.Error("Failed to create org", "err", err)
		return entities.Org{}, err
	}

	s.logger.Debug("Org created", "org_id", org.Id)
	s.logger.Debug("Creating API key")
	key := lib.GenerateId("sk")
	_, err = s.apiKeyRepository.Create(ctx, entities.ApiKey{
		OrgId:     org.Id,
		Id:        key,
		Key:       key,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to create API key", "org_id", org.Id, "err", err)
		return entities.Org{}, err
	}

	cohorts := []string{"signup_date"}
	for _, cohort := range cohorts {
		s.logger.Debugf("Creating cohort [%s]", cohort)
		_, err = s.cohortRepository.Create(ctx, entities.Cohort{
			OrgId:     org.Id,
			Id:        cohort,
			Name:      cohort,
			Type:      entities.CohortType(cohort),
			Metadata:  nil,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		})
		if err != nil {
			s.logger.Warn("Failed to create cohort", "org_id", org.Id, "cohort", cohort, "err", err)
		}
	}

	_ = s.pubsub.Publish(org.Id, topic.OrgCreated, org)

	return org, err
}
