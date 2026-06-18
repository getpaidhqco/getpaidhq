package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// settingRow is the postgres on-the-wire shape of a Setting. Package-internal.
//
// Note: `value` is a JSONB column but the domain Value is a plain `string`. The
// gorm adapter stored it with serializer:json, i.e. the on-disk representation
// is the JSON-encoded form of the string (wrapped in quotes). jsonCol[string]
// reproduces that exactly: it json.Marshals the string on write and unmarshals
// it on read, so the round-trip matches the historical schema.
type settingRow struct {
	OrgId     string
	ParentId  string
	Id        string
	Type      string
	Value     jsonCol[string]
	CreatedAt time.Time
	UpdatedAt time.Time
}

const settingColumns = `org_id, parent_id, id, value_type, value, created_at, updated_at`

func (r *settingRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.ParentId, &r.Id, &r.Type, &r.Value, &r.CreatedAt, &r.UpdatedAt)
}

func (r settingRow) toDomain() domain.Setting {
	return domain.Setting{
		OrgId:     r.OrgId,
		ParentId:  r.ParentId,
		Id:        r.Id,
		Type:      r.Type,
		Value:     r.Value.V,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func settingRowFromDomain(s domain.Setting) settingRow {
	return settingRow{
		OrgId:     s.OrgId,
		ParentId:  s.ParentId,
		Id:        s.Id,
		Type:      s.Type,
		Value:     newJSON(s.Value),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
