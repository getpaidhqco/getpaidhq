package domain

import "time"

type ApiKey struct {
	OrgId     string    `json:"org_id" validate:"required"`
	Id        string    `json:"id" validate:"required"`
	Key       string    `json:"key" validate:"required"`
	CreatedAt time.Time `json:"created_at" validate:"required"`
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}
