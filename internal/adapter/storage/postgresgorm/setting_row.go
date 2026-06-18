package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// settingRow is the postgres on-the-wire shape of a Setting. Package-internal.
//
// Note: `value` is stored with serializer:json even though it's a `string`,
// which means the on-disk representation is the JSON-encoded form of the
// string (i.e. wrapped in quotes). Preserved exactly as historical schema.
type settingRow struct {
	OrgId     string    `gorm:"column:org_id;primaryKey"`
	ParentId  string    `gorm:"column:parent_id;primaryKey"`
	Id        string    `gorm:"column:id;primaryKey"`
	Type      string    `gorm:"column:value_type"`
	Value     string    `gorm:"column:value;serializer:json"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (settingRow) TableName() string { return "settings" }

func (r settingRow) toDomain() domain.Setting {
	return domain.Setting{
		OrgId:     r.OrgId,
		ParentId:  r.ParentId,
		Id:        r.Id,
		Type:      r.Type,
		Value:     r.Value,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func settingRowsToDomain(rows []settingRow) []domain.Setting {
	out := make([]domain.Setting, len(rows))
	for i, r := range rows {
		out[i] = r.toDomain()
	}
	return out
}

func settingRowFromDomain(s domain.Setting) settingRow {
	return settingRow{
		OrgId:     s.OrgId,
		ParentId:  s.ParentId,
		Id:        s.Id,
		Type:      s.Type,
		Value:     s.Value,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
