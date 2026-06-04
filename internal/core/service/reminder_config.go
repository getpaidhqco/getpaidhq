package service

import (
	"context"
	"errors"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// ReminderConfigService satisfies the narrow ReminderConfigResolver port the
// billing sweep depends on.
var _ port.ReminderConfigResolver = (*ReminderConfigService)(nil)

type ReminderConfigService struct {
	settings port.SettingRepository
	logger   port.Logger
}

func NewReminderConfigService(settings port.SettingRepository, logger port.Logger) *ReminderConfigService {
	return &ReminderConfigService{settings: settings, logger: logger}
}

// ResolveReminderConfig returns the org's reminder policy, or the default when
// no setting exists (mirrors DunningService.ResolveConfig).
func (s *ReminderConfigService) ResolveReminderConfig(ctx context.Context, orgId string) (domain.ReminderConfig, error) {
	setting, err := s.settings.FindById(ctx, orgId, domain.ReminderConfigSettingParent, domain.ReminderConfigSettingId)
	if err != nil {
		// Not-found → default. The postgres FindById wraps gorm's
		// ErrRecordNotFound as port.ErrNotFound (see translateErr).
		if errors.Is(err, port.ErrNotFound) {
			return domain.DefaultReminderConfig(), nil
		}
		s.logger.Error("ResolveReminderConfig failed, using default", "orgId", orgId, "err", err.Error())
		return domain.DefaultReminderConfig(), nil
	}
	cfg, err := domain.ParseReminderConfig(setting.Value)
	if err != nil {
		s.logger.Error("invalid reminder config, using default", "orgId", orgId, "err", err.Error())
		return domain.DefaultReminderConfig(), nil
	}
	return cfg, nil
}

// SetReminderConfig upserts the org's reminder policy.
func (s *ReminderConfigService) SetReminderConfig(ctx context.Context, orgId string, cfg domain.ReminderConfig) error {
	value, err := cfg.Marshal()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = s.settings.Upsert(ctx, domain.Setting{
		OrgId:     orgId,
		ParentId:  domain.ReminderConfigSettingParent,
		Id:        domain.ReminderConfigSettingId,
		Type:      "json",
		Value:     value,
		// CreatedAt is inert on conflict: Upsert's DoUpdates omits created_at, so
		// the DB keeps the original — the column is effectively immutable after
		// the first write.
		CreatedAt: now,
		UpdatedAt: now,
	})
	return err
}
