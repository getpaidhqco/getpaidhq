package postgrespgx

import (
	"encoding/json"
	"time"

	"getpaidhq/internal/core/domain"
)

// pspConfigRow is the postgres on-the-wire shape of a PspConfig. Note the table
// name is `gateways` (legacy schema name).
//
// config and credentials are plain NOT NULL TEXT columns (default ”), not JSON
// columns — so they are scanned as strings here. Config is the JSON-encoded
// non-secret settings map; Credentials is the AES-GCM envelope sealed by the
// service layer (opaque here). The map<->JSON-string conversion is handled
// here at the boundary.
type pspConfigRow struct {
	OrgId       string
	Id          string
	PspId       string
	Name        string
	Active      bool
	Config      string
	Credentials string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const pspConfigColumns = `org_id, id, psp_id, name, active, config, credentials, created_at, updated_at`

func (r *pspConfigRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.PspId, &r.Name, &r.Active,
		&r.Config, &r.Credentials, &r.CreatedAt, &r.UpdatedAt)
}

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
		PspId:                domain.Gateway(r.PspId),
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
		PspId:       string(p.PspId),
		Name:        p.Name,
		Active:      p.Active,
		Config:      config,
		Credentials: p.EncryptedCredentials,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
