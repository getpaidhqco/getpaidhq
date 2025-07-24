package entities

import (
	"time"
)

type PaymentLink struct {
	OrgId     string    `json:"org_id"`
	Id        string    `json:"id"`
	Slug      string    `json:"slug"`
	Data      []byte    `json:"data"`
	Config    []byte    `json:"config"`
	SingleUse bool      `json:"single_use"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UsedAt    time.Time `json:"used_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}