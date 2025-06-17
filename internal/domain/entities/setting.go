package entities

import "time"

type Setting struct {
	OrgId     string    `json:"org_id"`
	ParentId  string    `json:"parent_id"`
	Id        string    `json:"id"`
	Type      string    `json:"value_type"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
