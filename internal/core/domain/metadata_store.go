package domain

import "time"

// MetadataStore is a generic key/value sidetable scoped to an Org and a parent
// of arbitrary type (orders, customers, etc.). Used for ad-hoc tagging and
// external-id lookups that don't belong on the parent's own row.
type MetadataStore struct {
	OrgId      string
	ParentId   string
	ParentType string
	Key        string
	Value      string
	Namespace  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
