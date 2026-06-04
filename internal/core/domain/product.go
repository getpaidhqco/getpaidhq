package domain

import "time"

// Product is the merchandise aggregate root. Cross-aggregate references are
// by ID only — Variants are loaded via service.ProductDetails composition.
type Product struct {
	OrgId       string
	Id          string
	Name        string
	Description string
	Metadata    map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
