package entities

import (
	"time"
)

type Variant struct {
	OrgId       string            `json:"org_id"`
	Id          string            `json:"id"`
	ProductId   string            `json:"product_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	Prices      []Price           `json:"prices"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type CreateVariantInput struct {
	OrgId       string            `json:"org_id"`
	Id          string            `json:"id"`
	ProductId   string            `json:"product_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}
