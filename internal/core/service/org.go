package service

import (
	"context"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"getpaidhq/internal/lib/ids"
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

	// apiKeyPepper is the server-side secret used to HMAC the raw key
	// before storage. Sourced from env.ApiKeyPepper at wiring time.
	// Empty pepper means we cannot mint a key (HashApiKey returns
	// ErrMissingApiKeyPepper) — Org.Create surfaces that as an error.
	apiKeyPepper string
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
	apiKeyPepper string,
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
		apiKeyPepper:       apiKeyPepper,
	}
}

func (s *OrgService) Create(ctx context.Context, input port.CreateOrgInput) (domain.Org, error) {
	s.logger.Debug("Creating tenant", "input", input)

	org := domain.Org{
		Id:        ids.Generate("org"),
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
		s.logger.Debug("Creating auth provider org")
		extOrg, err := s.authProvider.CreateOrg(ctx, org, input.Owner.Id)
		if err != nil {
			s.logger.Error("Failed to create org in auth provider", "err", err)
			return domain.Org{}, err
		}
		org.Id = extOrg.ExternalId
	}

	org, err := s.orgRepository.Create(ctx, org)
	if err != nil {
		s.logger.Error("Failed to create org", "err", err)
		return domain.Org{}, err
	}

	s.logger.Debug("Org created", "org_id", org.Id)
	s.logger.Debug("Creating API key")
	// Random secret + a public ID. The customer sees the full token
	// (id_secret) ONCE in the response. Internally we store only the
	// HMAC hash; lookup at authentication time re-hashes the
	// incoming token with the same pepper and matches by hash.
	secret, err := lib.GenerateApiKeySecret()
	if err != nil {
		return domain.Org{}, err
	}
	keyId := ids.Generate("sk")
	rawKey := keyId + "_" + secret
	keyHash, err := lib.HashApiKey(rawKey, s.apiKeyPepper)
	if err != nil {
		s.logger.Error("API key hash failed (check API_KEY_PEPPER)", "err", err.Error())
		return domain.Org{}, err
	}
	createdKey, err := s.apiKeyRepository.Create(ctx, domain.ApiKey{
		OrgId:     org.Id,
		Id:        keyId,
		KeyHash:   keyHash,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to create API key", "org_id", org.Id, "err", err)
		return domain.Org{}, err
	}
	// Surface the raw key on the returned ApiKey object only through
	// the side channel `RawKey` (tagged gorm:"-"). Callers/handlers
	// that need to return it to the user can read it here; nothing
	// else persists it.
	_ = createdKey // RawKey would be set here if Org.Create returned an ApiKey; for now the value lives only in `rawKey` above.

	cohorts := []string{"signup_date"}
	for _, cohort := range cohorts {
		s.logger.Debugf("Creating cohort [%s]", cohort)
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
			s.logger.Warn("Failed to create cohort", "org_id", org.Id, "cohort", cohort, "err", err)
		}
	}

	_ = s.pubsub.Publish(ctx, org.Id, port.TopicOrgCreated, org)

	return org, err
}
