package postgres

import (
	"encoding/json"
	"time"

	"getpaidhq/internal/core/domain"
)

// pspConfigRow is the postgres on-the-wire shape of a PspConfig. Note the
// table name is `gateways` (legacy schema name).
type pspConfigRow struct {
	OrgId  string         `gorm:"column:org_id;primaryKey"`
	Id     string         `gorm:"column:id;primaryKey"`
	PspId  domain.Gateway `gorm:"column:psp_id"`
	Name   string         `gorm:"column:name"`
	Active bool           `gorm:"column:active"`
	// Config is the non-secret settings JSON object; Credentials is the
	// AES-GCM envelope sealed by the service layer (opaque here).
	Config      string    `gorm:"column:config"`
	Credentials string    `gorm:"column:credentials"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (pspConfigRow) TableName() string { return "gateways" }

func (r pspConfigRow) toDomain() domain.PspConfig {
	var config map[string]string
	if r.Config != "" {
		// A corrupt value surfaces as a nil map; the adapter validates the
		// fields it needs, so this fails loudly there rather than here.
		_ = json.Unmarshal([]byte(r.Config), &config)
	}
	return domain.PspConfig{
		OrgId:                r.OrgId,
		Id:                   r.Id,
		PspId:                r.PspId,
		Name:                 r.Name,
		Active:               r.Active,
		Config:               config,
		EncryptedCredentials: r.Credentials,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func pspConfigRowFromDomain(p domain.PspConfig) pspConfigRow {
	config := ""
	if len(p.Config) > 0 {
		b, _ := json.Marshal(p.Config)
		config = string(b)
	}
	return pspConfigRow{
		OrgId:       p.OrgId,
		Id:          p.Id,
		PspId:       p.PspId,
		Name:        p.Name,
		Active:      p.Active,
		Config:      config,
		Credentials: p.EncryptedCredentials,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
