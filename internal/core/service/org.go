package service

import (
	"context"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
	"time"
)

type OrgService struct {
	pubsub             port.PubSub
	authProvider       port.AuthProvider
	orgRepository      port.OrgRepository
	customerRepository port.CustomerRepository
	apiKeyRepository   port.ApiKeyRepository
	settingRepository  port.SettingRepository
	metadataRepository port.MetadataStoreRepository
	logger             port.Logger
}

func NewOrgService(
	repo port.OrgRepository,
	pubsub port.PubSub,
	authProvider port.AuthProvider,
	customerRepository port.CustomerRepository,
	settingRepository port.SettingRepository,
	metadataRepository port.MetadataStoreRepository,
	apiKeyRepository port.ApiKeyRepository,
	logger port.Logger,
) *OrgService {
	return &OrgService{
		authProvider:       authProvider,
		orgRepository:      repo,
		pubsub:             pubsub,
		customerRepository: customerRepository,
		settingRepository:  settingRepository,
		apiKeyRepository:   apiKeyRepository,
		metadataRepository: metadataRepository,
		logger:             logger,
	}
}

func (s *OrgService) Create(ctx context.Context, input port.CreateOrgInput) (domain.Org, error) {
	s.logger.Debug("creating tenant", "input", input)

	org := domain.Org{
		Id:        lib.GenerateId("org"),
		Name:      input.Name,
		Status:    domain.OrgStatusActive,
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
		s.logger.Debug("creating auth provider org")
		extOrg, err := s.authProvider.CreateOrg(ctx, org, input.Owner.Id)
		if err != nil {
			s.logger.Error("failed to create org in auth provider", "error", err)
			return domain.Org{}, err
		}
		org.Id = extOrg.ExternalId
	}

	org, err := s.orgRepository.Create(ctx, org)
	if err != nil {
		s.logger.Error("failed to create org", "error", err)
		return domain.Org{}, err
	}

	s.logger.Debug("org created", "orgId", org.Id)
	s.logger.Debug("creating api key")
	key := lib.GenerateId("sk")
	_, err = s.apiKeyRepository.Create(ctx, domain.ApiKey{
		OrgId:     org.Id,
		Id:        key,
		Key:       key,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		s.logger.Error("failed to create api key", "orgId", org.Id, "error", err)
		return domain.Org{}, err
	}

	cohorts := []string{"signup_date"}
	for _, cohort := range cohorts {
		s.logger.Debug("creating cohort", "cohort", cohort)
		_, err = s.customerRepository.CreateCohort(ctx, domain.Cohort{
			OrgId:     org.Id,
			Id:        cohort,
			Name:      cohort,
			Type:      domain.CohortType(cohort),
			Metadata:  nil,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		})
		if err != nil {
			s.logger.Warn("failed to create cohort", "orgId", org.Id, "cohort", cohort, "error", err)
		}
	}

	_ = s.pubsub.Publish(org.Id, port.TopicOrgCreated, org)

	return org, err
}
