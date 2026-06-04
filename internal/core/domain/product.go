package domain

import "time"

// Product is the merchandise aggregate root. Variants is populated by the
// repo when a Preload-equivalent is used; for code paths that don't hydrate
// it, only the entity's own fields are reliable.
type Product struct {
	OrgId       string
	Id          string
	Name        string
	Description string
	Variants    []Variant
	Metadata    map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
