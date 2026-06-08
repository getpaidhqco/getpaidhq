package domain

import "time"

// ProductStatus is a Product's lifecycle. It has exactly two values; "sellable"
// is 1:1 with active. There is deliberately no "draft" or soft-deleted state —
// archiving is the retirement mechanism (see CONTEXT.md / ADR 0005).
type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "active"
	ProductStatusArchived ProductStatus = "archived"
)

// Product is the merchandise aggregate root. Cross-aggregate references are
// by ID only — Variants are loaded via service.ProductDetails composition.
type Product struct {
	OrgId       string
	Id          string
	Name        string
	Description string
	Status      ProductStatus
	// ArchivedAt is set when the product is archived and nil while active. It is
	// audit/report metadata only; Status is the queryable source of truth.
	ArchivedAt *time.Time
	Metadata   map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// IsArchived reports whether the product is retired and therefore not sellable.
func (p Product) IsArchived() bool { return p.Status == ProductStatusArchived }
