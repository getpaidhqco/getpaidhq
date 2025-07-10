package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Meter struct {
	OrgId           string                 `json:"org_id"`
	Id              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     pgtype.Text            `json:"description"`

	// Event Configuration
	EventName       string                 `json:"event_name"`
	EventFilter     map[string]interface{} `json:"event_filter"`

	// Aggregation Configuration
	AggregationType string                 `json:"aggregation_type"`
	ValueProperty   pgtype.Text            `json:"value_property"`

	// Display Configuration
	UnitType        string                 `json:"unit_type"`
	DisplayName     string                 `json:"display_name"`

	// Window Configuration
	WindowSize      pgtype.Text            `json:"window_size"`
	ResetInterval   pgtype.Text            `json:"reset_interval"`

	Metadata        map[string]string      `json:"metadata"`
	CreatedAt       pgtype.Timestamp       `json:"created_at"`
	UpdatedAt       pgtype.Timestamp       `json:"updated_at"`
}

func (m *Meter) ToEntity() entities.Meter {
	return entities.Meter{
		OrgId:           m.OrgId,
		Id:              m.Id,
		Name:            m.Name,
		Description:     m.Description.String,
		EventName:       m.EventName,
		EventFilter:     m.EventFilter,
		AggregationType: entities.AggregationType(m.AggregationType),
		ValueProperty:   m.ValueProperty.String,
		UnitType:        entities.UnitType(m.UnitType),
		DisplayName:     m.DisplayName,
		WindowSize:      m.WindowSize.String,
		ResetInterval:   m.ResetInterval.String,
		Metadata:        m.Metadata,
		CreatedAt:       m.CreatedAt.Time,
		UpdatedAt:       m.UpdatedAt.Time,
	}
}

func MeterFromEntity(entity entities.Meter) Meter {
	return Meter{
		OrgId:           entity.OrgId,
		Id:              entity.Id,
		Name:            entity.Name,
		Description:     pgtype.Text{String: entity.Description, Valid: entity.Description != ""},
		EventName:       entity.EventName,
		EventFilter:     entity.EventFilter,
		AggregationType: string(entity.AggregationType),
		ValueProperty:   pgtype.Text{String: entity.ValueProperty, Valid: entity.ValueProperty != ""},
		UnitType:        string(entity.UnitType),
		DisplayName:     entity.DisplayName,
		WindowSize:      pgtype.Text{String: entity.WindowSize, Valid: entity.WindowSize != ""},
		ResetInterval:   pgtype.Text{String: entity.ResetInterval, Valid: entity.ResetInterval != ""},
		Metadata:        entity.Metadata,
		CreatedAt:       pgtype.Timestamp{Time: entity.CreatedAt, Valid: !entity.CreatedAt.IsZero()},
		UpdatedAt:       pgtype.Timestamp{Time: entity.UpdatedAt, Valid: !entity.UpdatedAt.IsZero()},
	}
}