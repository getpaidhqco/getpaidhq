package domain

import "time"

// Setting is a key/value pair scoped to an Org and an opaque parent (e.g. an
// org-level setting has ParentId == OrgId; per-tenant settings reference a
// different parent). Value is a JSON-encoded string in the persistence layer.
type Setting struct {
	OrgId     string
	ParentId  string
	Id        string
	Type      string
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
