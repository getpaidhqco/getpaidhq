package domain

import "time"

// Variant is a sellable form of a Product. Cross-aggregate references are
// by ID only — Prices are loaded via service.VariantDetails composition.
type Variant struct {
	OrgId       string
	Id          string
	ProductId   string
	Name        string
	Description string
	Metadata    map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
