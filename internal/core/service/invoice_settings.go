package service

import (
	"context"
	"errors"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type InvoiceSettingsService struct {
	settings port.SettingRepository
	logger   port.Logger
}

func NewInvoiceSettingsService(settings port.SettingRepository, logger port.Logger) *InvoiceSettingsService {
	return &InvoiceSettingsService{settings: settings, logger: logger}
}

// ResolveInvoiceSettings returns the org's invoice settings, or the default when
// no setting exists (mirrors ReminderConfigService.ResolveReminderConfig).
func (s *InvoiceSettingsService) ResolveInvoiceSettings(ctx context.Context, orgId string) (domain.InvoiceSettings, error) {
	setting, err := s.settings.FindById(ctx, orgId, domain.InvoiceSettingsSettingParent, domain.InvoiceSettingsSettingId)
	if err != nil {
		// Not-found → default. The postgres FindById wraps gorm's
		// ErrRecordNotFound as port.ErrNotFound (see translateErr).
		if errors.Is(err, port.ErrNotFound) {
			return domain.DefaultInvoiceSettings(), nil
		}
		s.logger.Error("ResolveInvoiceSettings failed, using default", "orgId", orgId, "err", err.Error())
		return domain.DefaultInvoiceSettings(), nil
	}
	cfg, err := domain.ParseInvoiceSettings(setting.Value)
	if err != nil {
		s.logger.Error("invalid invoice settings, using default", "orgId", orgId, "err", err.Error())
		return domain.DefaultInvoiceSettings(), nil
	}
	return cfg, nil
}

// SetInvoiceSettings upserts the org's invoice settings.
func (s *InvoiceSettingsService) SetInvoiceSettings(ctx context.Context, orgId string, cfg domain.InvoiceSettings) error {
	value, err := cfg.Marshal()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = s.settings.Upsert(ctx, domain.Setting{
		OrgId:    orgId,
		ParentId: domain.InvoiceSettingsSettingParent,
		Id:       domain.InvoiceSettingsSettingId,
		Type:     "json",
		Value:    value,
		// CreatedAt is inert on conflict: Upsert's DoUpdates omits created_at, so
		// the DB keeps the original — the column is effectively immutable after
		// the first write.
		CreatedAt: now,
		UpdatedAt: now,
	})
	return err
}
