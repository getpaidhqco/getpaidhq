package domain

import "time"

type MetadataStore struct {
	OrgId      string    `json:"org_id"`
	ParentId   string    `json:"parent_id"`
	ParentType string    `json:"parent_type"`
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	Namespace  string    `json:"namespace"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
