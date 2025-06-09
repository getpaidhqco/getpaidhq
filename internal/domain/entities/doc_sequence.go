package entities

import (
	"time"
)

type DocSequence struct {
	OrgId     string    `json:"org_id"`
	Id        string    `json:"id"`
	Type      string    `json:"type"`
	Value     int       `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}