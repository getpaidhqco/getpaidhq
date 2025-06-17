package interfaces

import (
	"context"
	"payloop/internal/domain/entities"
)

type SettingsService interface {
	// GetSetting retrieves a setting and unmarshals its value into the provided type
	GetSetting(ctx context.Context, orgId string, parentId string, id string, result interface{}) error

	// GetSettingRaw retrieves a setting without unmarshaling its value
	GetSettingRaw(ctx context.Context, orgId string, parentId string, id string) (interface{}, error)

	// ListSettings retrieves all settings for a given organization and parent
	ListSettings(ctx context.Context, orgId string, parentId string) ([]entities.Setting, error)

	// CreateSetting creates a new setting
	CreateSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error)

	// UpdateSetting updates an existing setting
	UpdateSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error)

	// UpsertSetting creates a new setting if it doesn't exist or updates an existing setting by merging values
	UpsertSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error)

	// DeleteSetting deletes a setting
	DeleteSetting(ctx context.Context, orgId string, parentId string, id string) error
}
