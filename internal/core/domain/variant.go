package domain

import "time"

// Variant is a sellable form of a Product. Prices is populated by the repo
// when a Preload-equivalent is used; for code paths that don't hydrate it,
// only the entity's own fields are reliable.
type Variant struct {
	OrgId       string
	Id          string
	ProductId   string
	Name        string
	Description string
	Metadata    map[string]string
	Prices      []Price
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
