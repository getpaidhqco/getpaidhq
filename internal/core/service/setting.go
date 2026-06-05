package service

import (
	"context"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// SettingService is generic CRUD over org-scoped key/value settings. A setting is
// keyed by (OrgId, ParentId, Id); ParentId scopes a group (e.g. a gateway or a
// feature area), Id is the key within it. Value is an opaque string (often JSON).
type SettingService struct {
	settingRepository port.SettingRepository
	logger            port.Logger
}

func NewSettingService(settingRepository port.SettingRepository, logger port.Logger) *SettingService {
	return &SettingService{settingRepository: settingRepository, logger: logger}
}

// CreateSettingInput is the command for creating a setting.
type CreateSettingInput struct {
	OrgId    string
	ParentId string
	Id       string
	Type     string
	Value    string
}

func (s *SettingService) Create(ctx context.Context, in CreateSettingInput) (domain.Setting, error) {
	if in.Id == "" {
		return domain.Setting{}, lib.NewCustomError(lib.BadRequestError, "id is required", nil)
	}
	now := time.Now().UTC()
	return s.settingRepository.Create(ctx, domain.Setting{
		OrgId:     in.OrgId,
		ParentId:  in.ParentId,
		Id:        in.Id,
		Type:      in.Type,
		Value:     in.Value,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *SettingService) Get(ctx context.Context, orgId, parentId, id string) (domain.Setting, error) {
	return s.settingRepository.FindById(ctx, orgId, parentId, id)
}

func (s *SettingService) List(ctx context.Context, orgId, parentId string, p domain.Pagination) ([]domain.Setting, int, error) {
	return s.settingRepository.List(ctx, orgId, parentId, p)
}

// Upsert creates or replaces a setting by its key (the update path).
func (s *SettingService) Upsert(ctx context.Context, in CreateSettingInput) (domain.Setting, error) {
	if in.Id == "" {
		return domain.Setting{}, lib.NewCustomError(lib.BadRequestError, "id is required", nil)
	}
	now := time.Now().UTC()
	return s.settingRepository.Upsert(ctx, domain.Setting{
		OrgId:     in.OrgId,
		ParentId:  in.ParentId,
		Id:        in.Id,
		Type:      in.Type,
		Value:     in.Value,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *SettingService) Delete(ctx context.Context, orgId, parentId, id string) error {
	return s.settingRepository.Delete(ctx, orgId, parentId, id)
}
