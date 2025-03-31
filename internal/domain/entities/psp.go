package entities

import (
	"payloop/internal/domain/common"
	"time"
)

type PaymentServiceProvider struct {
	OrgId     string         `json:"org_id"`
	Id        string         `json:"id"`
	PspId     common.Gateway `json:"psp_id"`
	Name      string         `json:"name"`
	Active    bool           `json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
