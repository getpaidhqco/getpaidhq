package domain

import "time"

type Product struct {
	OrgId       string            `json:"org_id"`
	Id          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Variants    []Variant         `json:"variants"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
